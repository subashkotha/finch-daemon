// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	cerrdefs "github.com/containerd/errdefs"
	ncTypes "github.com/containerd/nerdctl/v2/pkg/api/types"

	"github.com/runfinch/finch-daemon/pkg/errdefs"
)

func (s *service) Pause(ctx context.Context, cid string, options ncTypes.ContainerPauseOptions) error {
	cont, err := s.getContainer(ctx, cid)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return errdefs.NewNotFound(err)
		}
		return err
	}
	status := s.client.GetContainerStatus(ctx, cont)
	if status != containerd.Running {
		if status == containerd.Paused {
			return errdefs.NewConflict(fmt.Errorf("container %s is already paused", cid))
		}
		return errdefs.NewConflict(fmt.Errorf("container %s is not running", cid))
	}

	return s.nctlContainerSvc.PauseContainer(ctx, cid, options)
}
