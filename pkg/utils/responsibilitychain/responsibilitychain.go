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
 * Create: 2021-09-01
 * Description: tools for chain of Responsibility
 ******************************************************************************/
package responsibilitychain

type Responsibility interface {
	Execute() error
	SetNexter(Responsibility)
	Nexter() Responsibility
}

func RunChainOfResponsibility(res Responsibility) error {
	if res == nil {
		return nil
	}
	if err := res.Execute(); err != nil {
		return err
	}
	nexter := res.Nexter()
	for nexter != nil {
		if err := nexter.Execute(); err != nil {
			return err
		}
		nexter = nexter.Nexter()
	}

	return nil
}
