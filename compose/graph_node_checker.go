/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compose

import (
	"fmt"
)

type nodeChecker func(nodeKey string, node *graphNode) error

func baseNodeChecker(nodeKey string, node *graphNode) error {

	if node.executorMeta.component == ComponentOfPassthrough && len(node.nodeInfo.inputKey) > 0 {
		return fmt.Errorf("paasthrough cannot be set input key, nodeKey=%v", nodeKey)
	}

	return nil
}

func nodeCheckerOfForbidProcessor(next nodeChecker) nodeChecker {
	return func(nodeKey string, node *graphNode) error {
		if node.nodeInfo.preProcessor != nil || node.nodeInfo.postProcessor != nil {
			return fmt.Errorf("only StateGraph support pre/post processor, nodeKey=%v", nodeKey)
		}
		return next(nodeKey, node)
	}
}

func nodeCheckerOfForbidNodeKey(next nodeChecker) nodeChecker {
	return func(nodeKey string, node *graphNode) error {
		if node.nodeInfo.key != "" {
			return fmt.Errorf("only Chain support WithNodeKey(), nodeKey=%v", nodeKey)
		}
		return next(nodeKey, node)
	}
}
