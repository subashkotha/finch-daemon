// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"io"

	ncTypes "github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/moby/moby/pkg/stdcopy"

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
	inStream, outStream, errStream, _, printSuccessResp, err := opts.GetStreams()
	if err != nil {
		return err
	}

	// if the caller wants neither to stream nor to view logs, return nothing
	if !opts.Stream && !opts.Logs {
		printSuccessResp()
		return nil
	}

	if opts.MuxStreams {
		errStream = stdcopy.NewStdWriter(errStream, stdcopy.Stderr)
		outStream = stdcopy.NewStdWriter(outStream, stdcopy.Stdout)
	}

	// Determine which streams to use based on flags
	var (
		stdin          io.Reader
		stdout, stderr io.Writer
	)
	if opts.UseStdin {
		stdin = inStream
	}
	if opts.UseStdout {
		stdout = outStream
	}
	if opts.UseStderr {
		stderr = errStream
	}

	// Send success response before streaming begins, as per Docker's behavior
	printSuccessResp()

	if opts.Logs {
		since := "0s"
		if opts.Stream {
			since = ""
		}

		logOpts := ncTypes.ContainerLogsOptions{
			Stdout:     stdout,
			Stderr:     stderr,
			GOptions:   ncTypes.GlobalCommandOptions{},
			Follow:     opts.Stream,
			Timestamps: false,
			Tail:       0,
			Since:      since,
			Until:      "",
		}
		err = s.nctlContainerSvc.Logs(ctx, cid, logOpts)
		if err != nil {
			s.logger.Debugf("failed to stream logs for container %s: %v", cid, err)
			return err
		}
		return nil
	}

	attachOpts := ncTypes.ContainerAttachOptions{
		Stdin:      stdin,
		Stdout:     stdout,
		Stderr:     stderr,
		DetachKeys: opts.DetachKeys,
		GOptions:   ncTypes.GlobalCommandOptions{},
	}
	err = s.nctlContainerSvc.AttachContainer(ctx, cid, attachOpts)
	if err != nil {
		s.logger.Debugf("failed to attach to container %s: %v", cid, err)
		return err
	}
	return nil
}
