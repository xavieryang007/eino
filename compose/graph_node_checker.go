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
