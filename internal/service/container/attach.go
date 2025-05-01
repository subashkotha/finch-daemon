// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	ncTypes "github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/runfinch/finch-daemon/api/types"
)

// Attach attaches stdin, stdout and stderr to the container using nerdctl attach
func (s *service) Attach(ctx context.Context, cid string, opts *types.AttachOptions) error {
	// fetch container
	con, err := s.getContainer(ctx, cid)
	if err != nil {
		return err
	}
	s.logger.Debugf("attaching container: %s", con.ID())

	// set up io streams
	stdin, stdout, stderr, _, printSuccessResp, err := opts.GetStreams()
	if err != nil {
		return err
	}

	// Print success response before attaching
	printSuccessResp()

	// If logs is requested, we'll use nerdctl logs instead of attach
	if opts.Logs {
		logsOpts := ncTypes.ContainerLogsOptions{
			Stdout:     stdout,
			Stderr:     stderr,
			Follow:     opts.Stream,
			Timestamps: false,
			Tail:       0, // 0 means all logs
		}
		return s.nctlContainerSvc.Logs(ctx, cid, logsOpts)
	}

	// Convert to nerdctl's attach options
	nerdctlOpts := ncTypes.ContainerAttachOptions{
		Stdin:      stdin,
		Stdout:     stdout,
		Stderr:     stderr,
		DetachKeys: opts.DetachKeys,
		GOptions:   ncTypes.GlobalCommandOptions{},
	}

	// Call nerdctl's attach implementation
	err = s.nctlContainerSvc.AttachContainer(ctx, cid, nerdctlOpts)
	if err != nil {
		s.logger.Debugf("failed to attach to the container: %s", cid)
		return err
	}

	return nil
}
