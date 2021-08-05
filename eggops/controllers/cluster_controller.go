/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	eggov1 "isula.org/eggo/eggops/api/v1"
)

const (
	ClusterFinalizerName = "cluster.eggo.isula.org/finalizer"
	MachineBindingFormat = "machinebind-%s"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=eggo.isula.org,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eggo.isula.org,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eggo.isula.org,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=eggo.isula.org,resources=machinebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eggo.isula.org,resources=machinebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1,resources=configmaps/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := log.FromContext(ctx)
	r.Log = log

	cluster := &eggov1.Cluster{}
	if terr := r.Get(ctx, req.NamespacedName, cluster); terr != nil {
		if client.IgnoreNotFound(terr) != nil {
			log.Error(terr, "unable to get cluster")
		}
		return ctrl.Result{}, client.IgnoreNotFound(terr)
	}
	// update cluster after Reconcile
	defer func() {
		if err != nil {
			return
		}
		if cluster.Status.Deleted {
			return
		}
		// TODO: maybe should use patch to replace
		if err = r.Update(ctx, cluster); err != nil {
			log.Error(err, "unable to update cluster", "name", cluster.Name)
			return
		}
		log.Info("update cluster success", "name", cluster.Name)
	}()
	log.Info(fmt.Sprintf("get cluster: %s", cluster.Name))

	if cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		// this cluster is not being deleted
		if !foundString(cluster.GetFinalizers(), ClusterFinalizerName) {
			// register our finalizer
			controllerutil.AddFinalizer(cluster, ClusterFinalizerName)
			return
		}
	} else {
		// this cluster is being deleting
		if foundString(cluster.GetFinalizers(), ClusterFinalizerName) {
			return r.reconcileDelete(ctx, cluster)
		}
		// Stop reconcile, because this cluster is deleted
		log.Info("cluster is being deleted", "name", cluster.Name)
		return
	}

	return r.reconcile(ctx, cluster)
}

func (r *ClusterReconciler) prepareDeleteClusterJob(ctx context.Context, cluster *eggov1.Cluster) (bool, error) {
	cmName := fmt.Sprintf(eggov1.ClusterConfigMapNameFormat, cluster.Name, "cmd-config")
	job := &batch.Job{}
	jobName := fmt.Sprintf("%s-delete-job", cluster.Name)
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: eggov1.EggoNamespaceName}, job)
	if err == nil {
		finish, terr := jobIsFinished(job)
		if finish {
			history := &eggov1.JobHistory{
				Name:      job.GetName(),
				StartTime: job.GetCreationTimestamp(),
			}
			if terr != nil {
				history.Message = terr.Error()
			} else {
				history.Message = "success"
			}
			background := metav1.DeletePropagationBackground
			if err = r.Delete(ctx, job, &client.DeleteOptions{PropagationPolicy: &background}); err == nil {
				cluster.Status.JobHistorys = append(cluster.Status.JobHistorys, history)
			}
		}
		return finish, terr
	}
	if client.IgnoreNotFound(err) != nil {
		return false, err
	}
	// if not found job, just create new job
	yamlPath := "/config/" + cluster.Name + ".yaml"
	//Command: []string{"eggo", "-d", "deploy", "-f", yamlPath},
	job = createEggoJobConfig(jobName, "eggo-delete-cluster", "eggo:"+eggov1.ImageVersion,
		yamlPath, cmName, []string{"sleep", "10"})
	err = r.Create(ctx, job)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (r *ClusterReconciler) reconcileDelete(ctx context.Context, cluster *eggov1.Cluster) (ctrl.Result, error) {
	log := r.Log
	// TODO: cleanup external resources

	// Step 1: delete running job of cluster
	if cluster.Status.JobRef != nil {
		job := &batch.Job{}
		err := r.Get(ctx, ReferenceToNamespacedName(cluster.Status.JobRef), job)
		if err == nil {
			finish, _ := jobIsFinished(job)
			// delete old job
			background := metav1.DeletePropagationBackground
			if !finish {
				var graceSec int64 = 60
				err = r.Delete(ctx, job, &client.DeleteOptions{GracePeriodSeconds: &graceSec, PropagationPolicy: &background})
			} else {
				err = r.Delete(ctx, job, &client.DeleteOptions{PropagationPolicy: &background})
			}
			if err != nil {
				log.Error(err, "delete running job for cluster")
			}
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}

		if client.IgnoreNotFound(err) != nil {
			r.Log.Error(err, "get running job failed")
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}

		r.Log.Info("delete running job success")
		cluster.Status.JobRef = nil
	}

	// Step 2: run job to delete cluster
	if cluster.IsCreated() {
		finish, err := r.prepareDeleteClusterJob(ctx, cluster)
		if !finish {
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		// delete cluster success, just update status of cluster
		cluster.Status.HasCluster = false
	}

	// Step 3: delete machinebinding
	if cluster.Status.MachineBindingRef != nil {
		var mb eggov1.MachineBinding
		err := r.Get(ctx, ReferenceToNamespacedName(cluster.Status.MachineBindingRef), &mb)
		if err == nil {
			r.Delete(ctx, &mb)
			log.Info("ignore err: delete machine binding for cluster")
			return ctrl.Result{Requeue: true}, nil
		}
		log.Info("delete machine binding success...")
		cluster.Status.MachineBindingRef = nil
	}

	// Step 4: delete configmap
	if cluster.Status.ConfigRef != nil {
		var cm v1.ConfigMap
		err := r.Get(ctx, ReferenceToNamespacedName(cluster.Status.ConfigRef), &cm)
		if err == nil {
			r.Delete(ctx, &cm)
			log.Info("ignore err: delete configmap for cluster")
			return ctrl.Result{Requeue: true}, nil
		}
		log.Info("delete configmap success...")
		cluster.Status.ConfigRef = nil
	}

	cluster.Status.Deleted = true
	// Step 5: remove our finalizer, so we can remove cluster
	controllerutil.RemoveFinalizer(cluster, ClusterFinalizerName)
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) filterMachines(ctx context.Context, namespace string, config eggov1.RequireMachineConfig, usage int32) ([]eggov1.Machine, error) {
	var result []eggov1.Machine
	if config.Number <= 0 {
		return nil, nil
	}
	var tmp eggov1.MachineList
	labelSet := labels.Set{}
	for k, v := range config.Features {
		labelSet[k] = v
	}
	options := client.ListOptions{Namespace: namespace}
	options.LabelSelector = labels.SelectorFromSet(labelSet)
	if err := r.List(ctx, &tmp, &options); err != nil {
		return nil, err
	}
	requiredNum := int(config.Number)
	if len(tmp.Items) < requiredNum {
		return nil, fmt.Errorf("no enough machine for requires")
	}
	foundCnt := 0
	for _, item := range tmp.Items {
		mbSet := labels.Set{}
		var mbs eggov1.MachineBindingList
		mbOptions := client.ListOptions{}
		mbSet[item.Name] = ""
		mbOptions.LabelSelector = labels.SelectorFromSet(mbSet)
		mbOptions.Namespace = eggov1.EggoNamespaceName
		if err := r.List(ctx, &mbs, &mbOptions); err == nil && len(mbs.Items) > 0 {
			if u, ok := mbs.Items[0].Spec.Usages[string(item.GetUID())]; ok {
				if u&usage != 0 {
					// if machine is binding for usage, just skip it
					continue
				}
			}
		}
		result = append(result, item)
		foundCnt++
		if foundCnt == requiredNum {
			break
		}
	}
	if foundCnt != requiredNum {
		return nil, fmt.Errorf("no enough machine for requires")
	}

	return result, nil
}

func (r *ClusterReconciler) prepareMachineBinding(ctx context.Context, cluster *eggov1.Cluster) error {
	log := r.Log
	var mb eggov1.MachineBinding
	labels := make(map[string]string)
	// Step 1.1: get machines for master
	masterSelectedMachines, err := r.filterMachines(ctx, cluster.Namespace, cluster.Spec.MasterRequire, eggov1.UsageMaster)
	if err != nil {
		log.Error(err, "master requires")
		return err
	}
	log.Info(fmt.Sprintf("get machines for master: %v", eggov1.PrintMachineSlice(masterSelectedMachines)))
	for _, m := range masterSelectedMachines {
		mb.AddMachine(m, eggov1.UsageMaster)
		labels[m.Name] = ""
	}

	// Step 1.2: get machines for worker
	workerSelectedMachines, err := r.filterMachines(ctx, cluster.Namespace, cluster.Spec.WorkerRequire, eggov1.UsageWorker)
	if err != nil {
		log.Error(err, "worker requires")
		return err
	}
	log.Info(fmt.Sprintf("get machines for worker: %v", eggov1.PrintMachineSlice(workerSelectedMachines)))
	for _, m := range workerSelectedMachines {
		mb.AddMachine(m, eggov1.UsageWorker)
		labels[m.Name] = ""
	}

	// Step 1.3: get machines for etcd
	etcdSelectedMachines, err := r.filterMachines(ctx, cluster.Namespace, cluster.Spec.EtcdRequire, eggov1.UsageEtcd)
	if err != nil {
		log.Error(err, "etcd requires")
		return err
	}
	log.Info(fmt.Sprintf("get machines for etcd: %v", eggov1.PrintMachineSlice(etcdSelectedMachines)))
	for _, m := range etcdSelectedMachines {
		mb.AddMachine(m, eggov1.UsageEtcd)
		labels[m.Name] = ""
	}

	// Step 1.4: get machines for loadbalance
	lbSelectedMachines, err := r.filterMachines(ctx, cluster.Namespace, cluster.Spec.LoadbalanceRequires, eggov1.UsageLoadbalance)
	if err != nil {
		log.Error(err, "loadbalance requires")
		return err
	}
	log.Info(fmt.Sprintf("get machines for loadbalance: %v", eggov1.PrintMachineSlice(lbSelectedMachines)))
	for _, m := range lbSelectedMachines {
		mb.AddMachine(m, eggov1.UsageLoadbalance)
		labels[m.Name] = ""
	}

	mb.SetName(fmt.Sprintf(MachineBindingFormat, cluster.Name))
	mb.SetLabels(labels)
	mb.SetNamespace(eggov1.EggoNamespaceName)

	if err = r.Create(ctx, &mb); err != nil {
		log.Error(err, "create machine binding for cluster", "name", cluster.Name)
		return err
	}
	return nil
}

func (r *ClusterReconciler) prepareEggoConfig(ctx context.Context, cluster *eggov1.Cluster) (ctrl.Result, error) {
	res := ctrl.Result{}
	// configmap get machines from machine-binding;
	// maybe require new machine or remove machine before create configmap, just ignore them;
	// we will deal with them in join/cleanup
	var mb eggov1.MachineBinding
	err := r.Get(ctx, ReferenceToNamespacedName(cluster.Status.MachineBindingRef), &mb)
	if err != nil {
		r.Log.Error(err, "get machine binding for cluster config failed", "name", cluster.Name)
		return ctrl.Result{}, err
	}

	data, err := ConvertClusterToEggoConfig(cluster, &mb)
	if err != nil {
		r.Log.Error(err, "convert cluster failed", "name", cluster.Name)
		return res, err
	}

	cm := v1.ConfigMap{}
	cmName := fmt.Sprintf(eggov1.ClusterConfigMapNameFormat, cluster.Name, "cmd-config")
	err = r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: eggov1.EggoNamespaceName}, &cm)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return res, err
		}
		cm.SetName(cmName)
		cm.SetNamespace(eggov1.EggoNamespaceName)
		// owner reference cause to remove configmap
		cm.BinaryData = make(map[string][]byte)
		cm.BinaryData[eggov1.ClusterConfigMapBinaryConfKey] = data
		return ctrl.Result{RequeueAfter: time.Second * 2}, r.Create(ctx, &cm)
	}
	cluster.Status.ConfigRef, err = reference.GetReference(r.Scheme, &cm)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 2}, err
	}
	r.Log.Info("save cluster config into configmap success", "name", cluster.Name)
	return res, nil
}

func createEggoJobConfig(jobName, containerName, image, configPath, configMapName string, command []string) *batch.Job {
	return &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        jobName,
			Namespace:   eggov1.EggoNamespaceName,
		},
		Spec: batch.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  containerName,
							Image: image,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "cluster-config",
									MountPath: configPath,
									ReadOnly:  true,
								},
							},
							Command: command,
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
					Volumes: []v1.Volume{
						{
							Name: "cluster-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ClusterReconciler) prepareCreateClusterJob(ctx context.Context, cluster *eggov1.Cluster) error {
	cmName := fmt.Sprintf(eggov1.ClusterConfigMapNameFormat, cluster.Name, "cmd-config")
	job := &batch.Job{}
	jobName := fmt.Sprintf("%s-create-job", cluster.Name)
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: eggov1.EggoNamespaceName}, job)
	if err == nil {
		cluster.Status.JobRef, err = reference.GetReference(r.Scheme, job)
		if err != nil {
			r.Log.Error(err, "get reference for job failed")
		}
		return err
	}
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	// if not found job, just create new job
	yamlPath := "/config/" + cluster.Name + ".yaml"
	//Command: []string{"eggo", "-d", "deploy", "-f", yamlPath},
	job = createEggoJobConfig(jobName, "eggo-create-cluster", "eggo:"+eggov1.ImageVersion,
		yamlPath, cmName, []string{"sleep", "10"})
	err = r.Create(ctx, job)
	if err != nil {
		return err
	}

	return nil
}

func jobIsFinished(job *batch.Job) (bool, error) {
	for _, c := range job.Status.Conditions {
		if c.Status == v1.ConditionTrue {
			if c.Type == batch.JobComplete {
				return true, nil
			}
			if c.Type == batch.JobFailed {
				return true, fmt.Errorf("job: %s failed: %s", job.GetName(), c.Reason)
			}
		}
	}

	return false, nil
}

func (r *ClusterReconciler) checkAndLogClusterJob(ctx context.Context, cluster *eggov1.Cluster) (bool, error) {
	r.Log.Info("check job status")
	job := &batch.Job{}
	err := r.Get(ctx, ReferenceToNamespacedName(cluster.Status.JobRef), job)
	if err != nil {
		return false, err
	}
	var finish bool
	finish, err = jobIsFinished(job)
	if !finish {
		// just requeue to wait job finish
		return finish, err
	}

	history := &eggov1.JobHistory{
		Name:       job.GetName(),
		StartTime:  job.GetCreationTimestamp(),
		FinishTime: job.GetDeletionTimestamp(),
	}
	if err != nil {
		r.Log.Error(err, "create cluster job failed, remove job...")
		background := metav1.DeletePropagationBackground
		if terr := r.Delete(ctx, job, &client.DeleteOptions{PropagationPolicy: &background}); terr != nil {
			r.Log.Error(err, "delete create cluster job failed")
			return finish, err
		}
		r.Log.Info("delete old create cluster job success")

		history.Message = err.Error()
		cluster.Status.JobHistorys = append(cluster.Status.JobHistorys, history)
		// clear ref of failed job
		cluster.Status.JobRef = nil
	}

	return finish, err
}

func (r *ClusterReconciler) updateMachineBindingStatus(ctx context.Context, cluster *eggov1.Cluster) error {
	var mb eggov1.MachineBinding
	err := r.Get(ctx, ReferenceToNamespacedName(cluster.Status.MachineBindingRef), &mb)
	if err != nil {
		return err
	}
	for uid, usage := range mb.Spec.Usages {
		mb.UpdateCondition(eggov1.MachineCondition{UsagesStatus: usage, Message: "success"}, uid)
	}
	return r.Update(ctx, &mb)
}

func (r *ClusterReconciler) reconcileCreate(ctx context.Context, cluster *eggov1.Cluster) (res ctrl.Result, err error) {
	res = ctrl.Result{}
	// Step 1: get free machines which match feature of cluster required
	if cluster.Status.MachineBindingRef == nil {
		var mb eggov1.MachineBinding
		err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf(MachineBindingFormat, cluster.Name), Namespace: eggov1.EggoNamespaceName}, &mb)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				r.Log.Error(err, "get machine binding for cluster", "name", cluster.Name)
				return
			}
			err = r.prepareMachineBinding(ctx, cluster)
			if err != nil {
				r.Log.Error(err, "prepare machine binding for cluster", "name", cluster.Name)
			}
			// requeue to wait machine binding success
			return ctrl.Result{RequeueAfter: time.Second * 2}, err
		}

		cluster.Status.MachineBindingRef, err = reference.GetReference(r.Scheme, &mb)
		if err != nil {
			r.Log.Error(err, "unable to reference to machine binding for cluster", "name", cluster.Name)
			return
		}
	}

	// Step 2: save cluster config into configmap
	if cluster.Status.ConfigRef == nil {
		return r.prepareEggoConfig(ctx, cluster)
	}

	// Step 3: create job to create cluster
	if cluster.Status.JobRef == nil {
		// create job
		r.Log.Info("create job...")
		err = r.prepareCreateClusterJob(ctx, cluster)
		if err != nil {
			r.Log.Error(err, "prepare job to create cluster", "name", cluster.Name)
		}
		// requeue after prepare job
		return ctrl.Result{RequeueAfter: time.Second * 2}, err
	}

	// Step 4: wait job success
	finish, err := r.checkAndLogClusterJob(ctx, cluster)
	if !finish || err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	}

	// Step 5: update status of resources, cluster and machinebinding
	// TODO: update other status
	err = r.updateMachineBindingStatus(ctx, cluster)
	if err != nil {
		return
	}
	cluster.Status.HasCluster = true
	cluster.Status.Message = "create cluster job successfully"

	r.Log.Info("create new cluster success", "name", cluster.Name)
	return
}

func foundString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func (r *ClusterReconciler) reconcile(ctx context.Context, cluster *eggov1.Cluster) (res ctrl.Result, err error) {
	log := r.Log
	res = ctrl.Result{}

	// create new cluster
	if !cluster.IsCreated() {
		res, err = r.reconcileCreate(ctx, cluster)
		if err != nil {
			log.Error(err, "unable to create cluster")
			return
		}
		// TODO: when need requeue
		return
	}

	// TODO: finish join, cleanup node and update cluster
	log.Info("call eggo job to join/cleanup node from cluster", "name", cluster.Name)

	return res, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eggov1.Cluster{}).
		Complete(r)
}
