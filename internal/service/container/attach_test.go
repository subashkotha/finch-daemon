// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	containerd "github.com/containerd/containerd/v2/client"
	ncTypes "github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/runfinch/finch-daemon/mocks/mocks_archive"

	"github.com/runfinch/finch-daemon/api/handlers/container"
	attachTypes "github.com/runfinch/finch-daemon/api/types"
	"github.com/runfinch/finch-daemon/mocks/mocks_backend"
	"github.com/runfinch/finch-daemon/mocks/mocks_container"
	"github.com/runfinch/finch-daemon/mocks/mocks_logger"
	"github.com/runfinch/finch-daemon/pkg/errdefs"
)

var _ = Describe("Container Attach API ", func() {
	var (
		ctx          context.Context
		mockCtrl     *gomock.Controller
		logger       *mocks_logger.Logger
		cdClient     *mocks_backend.MockContainerdClient
		ncClient     *mocks_backend.MockNerdctlContainerSvc
		tarExtractor *mocks_archive.MockTarExtractor
		service      container.Service

		mockWriter   *bytes.Buffer
		stopChannel  chan os.Signal
		setupStreams func() (io.Reader, io.Writer, io.Writer, chan os.Signal, func(), error)
		cid          string
	)
	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		logger = mocks_logger.NewLogger(mockCtrl)
		cdClient = mocks_backend.NewMockContainerdClient(mockCtrl)
		ncClient = mocks_backend.NewMockNerdctlContainerSvc(mockCtrl)
		tarExtractor = mocks_archive.NewMockTarExtractor(mockCtrl)

		service = NewService(cdClient, mockNerdctlService{ncClient, nil}, logger, nil, nil, tarExtractor)

		mockWriter = new(bytes.Buffer)
		stopChannel = make(chan os.Signal, 1)
		signal.Notify(stopChannel, syscall.SIGTERM, syscall.SIGINT)
		setupStreams = func() (io.Reader, io.Writer, io.Writer, chan os.Signal, func(), error) {
			return mockWriter, mockWriter, mockWriter, stopChannel, func() {}, nil
		}
		cid = "test-container"
	})
	Context("service", func() {
		It("should successfully attach to a container", func() {
			// set up mocks
			con := mocks_container.NewMockContainer(mockCtrl)
			cdClient.EXPECT().SearchContainer(gomock.Any(), gomock.Any()).Return([]containerd.Container{con}, nil)
			logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Return()
			con.EXPECT().ID().Return(cid)
			ncClient.EXPECT().AttachContainer(gomock.Any(), cid, gomock.Any()).Return(nil)

			// set up options
			opts := attachTypes.AttachOptions{
				GetStreams: setupStreams,
				UseStdin:   false,
				UseStdout:  true,
				UseStderr:  true,
				DetachKeys: "ctrl-p,ctrl-q",
				MuxStreams: true,
			}

			// run function and assertions
			err := service.Attach(ctx, cid, &opts)
			Expect(err).Should(BeNil())
		})

		It("should successfully attach with custom detach keys", func() {
			// set up mocks
			con := mocks_container.NewMockContainer(mockCtrl)
			cdClient.EXPECT().SearchContainer(gomock.Any(), gomock.Any()).Return([]containerd.Container{con}, nil)
			logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Return()
			con.EXPECT().ID().Return(cid)
			ncClient.EXPECT().AttachContainer(gomock.Any(), cid, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, opts interface{}) error {
					attachOpts := opts.(ncTypes.ContainerAttachOptions)
					Expect(attachOpts.DetachKeys).Should(Equal("ctrl-a,d"))
					return nil
				},
			).Return(nil)

			// set up options
			opts := attachTypes.AttachOptions{
				GetStreams: setupStreams,
				UseStdin:   true,
				UseStdout:  true,
				UseStderr:  true,
				DetachKeys: "ctrl-a,d",
				MuxStreams: true,
			}

			// run function and assertions
			err := service.Attach(ctx, cid, &opts)
			Expect(err).Should(BeNil())
		})

		It("should successfully attach with stdin only", func() {
			// set up mocks
			con := mocks_container.NewMockContainer(mockCtrl)
			cdClient.EXPECT().SearchContainer(gomock.Any(), gomock.Any()).Return([]containerd.Container{con}, nil)
			logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Return()
			con.EXPECT().ID().Return(cid)
			ncClient.EXPECT().AttachContainer(gomock.Any(), cid, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, opts interface{}) error {
					attachOpts := opts.(ncTypes.ContainerAttachOptions)
					Expect(attachOpts.Stdin).ShouldNot(BeNil())
					Expect(attachOpts.Stdout).Should(BeNil())
					Expect(attachOpts.Stderr).Should(BeNil())
					return nil
				},
			).Return(nil)

			// set up options
			opts := attachTypes.AttachOptions{
				GetStreams: setupStreams,
				UseStdin:   true,
				UseStdout:  false,
				UseStderr:  false,
				DetachKeys: "ctrl-p,ctrl-q",
				MuxStreams: false,
			}

			// run function and assertions
			err := service.Attach(ctx, cid, &opts)
			Expect(err).Should(BeNil())
		})

		It("should return an error if opts.GetStreams returns an error", func() {
			// set up expected mocks, errors and the setupstreams to return an error
			con := mocks_container.NewMockContainer(mockCtrl)
			cdClient.EXPECT().SearchContainer(gomock.Any(), gomock.Any()).Return([]containerd.Container{con}, nil)
			logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Return()
			con.EXPECT().ID().Return(cid)
			expErr := fmt.Errorf("error")
			setupStreams = func() (io.Reader, io.Writer, io.Writer, chan os.Signal, func(), error) {
				return nil, nil, nil, nil, nil, expErr
			}
			// set up options
			opts := attachTypes.AttachOptions{
				GetStreams: setupStreams,
				UseStdin:   true,
				UseStdout:  true,
				UseStderr:  true,
				DetachKeys: "ctrl-p,ctrl-q",
				MuxStreams: false,
			}

			// run function and assertions
			err := service.Attach(ctx, cid, &opts)
			Expect(err).Should(Equal(expErr))
		})

		It("should return a not found error if a container can't be found", func() {
			// set up mocks
			cdClient.EXPECT().SearchContainer(gomock.Any(), gomock.Any()).Return([]containerd.Container{}, nil)
			logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Return()

			// set up options
			opts := attachTypes.AttachOptions{
				GetStreams: setupStreams,
				UseStdin:   true,
				UseStdout:  true,
				UseStderr:  true,
				DetachKeys: "ctrl-p,ctrl-q",
				MuxStreams: true,
			}

			// run function and assertions
			err := service.Attach(ctx, cid, &opts)
			Expect(errdefs.IsNotFound(err)).Should(BeTrue())
		})

		It("should return an error if nerdctl attach fails", func() {
			// set up mocks
			con := mocks_container.NewMockContainer(mockCtrl)
			cdClient.EXPECT().SearchContainer(gomock.Any(), gomock.Any()).Return([]containerd.Container{con}, nil)
			logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Return().Times(2)
			con.EXPECT().ID().Return(cid)
			expErr := fmt.Errorf("nerdctl attach error")
			ncClient.EXPECT().AttachContainer(gomock.Any(), cid, gomock.Any()).Return(expErr)

			// set up options
			opts := attachTypes.AttachOptions{
				GetStreams: setupStreams,
				UseStdin:   false,
				UseStdout:  true,
				UseStderr:  true,
				DetachKeys: "ctrl-p,ctrl-q",
				MuxStreams: true,
			}

			// run function and assertions
			err := service.Attach(ctx, cid, &opts)
			Expect(err).Should(Equal(expErr))
		})
	})
})
