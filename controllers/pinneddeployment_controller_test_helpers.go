package controllers

import (
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
)

// Helper wrapper to allow inlining the build function without handling the error.
func replicaSetOnly(replicaSet *appsv1.ReplicaSet, err error) appsv1.ReplicaSet {
	if err != nil {
		panic(errors.Wrap(err, "error generating replicaset in test"))
	}
	return *replicaSet
}
