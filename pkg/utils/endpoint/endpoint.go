/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2021. All rights reserved.
 * eggo licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: haozi007
 * Create: 2021-05-20
 * Description: endpoint utils
 ******************************************************************************/
package endpoint

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	validation "k8s.io/apimachinery/pkg/util/validation"
)

func GetEndpoint(advertiseAddr string, bindPort int) (string, error) {
	if !ValidPort(bindPort) {
		return "", fmt.Errorf("invalid port: %d", bindPort)
	}

	if ip := net.ParseIP(advertiseAddr); ip == nil {
		errs := validation.IsDNS1123Subdomain(advertiseAddr)
		if len(errs) > 0 {
			return "", fmt.Errorf("invalid domain: '%s' for RFC-1123 subdomain", advertiseAddr)
		}
	}

	url := FormatURL(advertiseAddr, strconv.Itoa(bindPort))
	return url.String(), nil
}

func ValidPort(port int) bool {
	return (port >= 1 && port <= 65535)
}

func ParsePort(port string) (int, error) {
	tport, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return 0, err
	}
	if ValidPort(int(tport)) {
		return int(tport), nil
	}
	return 0, fmt.Errorf("invalid port: %s", port)
}

func FormatURL(host, port string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(host, port),
	}
}
