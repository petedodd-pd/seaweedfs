package weed_server

import (
	"context"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/filer"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/util"
	"github.com/seaweedfs/seaweedfs/weed/util/atime"
)

func (fs *FilerServer) TouchAccessTime(ctx context.Context, req *filer_pb.TouchAccessTimeRequest) (*filer_pb.TouchAccessTimeResponse, error) {
	candidate := time.Unix(0, req.ClientAtimeNs)
	if req.ClientAtimeNs == 0 {
		candidate = time.Now()
	}
	persistedNs, updated, err := fs.touchAccessTime(ctx, util.NewFullPath(req.Directory, req.Name), candidate)
	if err != nil {
		return &filer_pb.TouchAccessTimeResponse{}, err
	}
	return &filer_pb.TouchAccessTimeResponse{PersistedAtimeNs: persistedNs, Updated: updated}, nil
}

// touchAccessTime applies the configured atime policy to an entry and persists
// it through the filer store. Returns (persistedNs, updated, err) where
// persistedNs is 0 when no write was performed.
func (fs *FilerServer) touchAccessTime(ctx context.Context, fullpath util.FullPath, candidate time.Time) (int64, bool, error) {
	policy := fs.atimePolicy()
	if policy.Mode == filer.AtimeModeOff {
		return 0, false, nil
	}

	entry, err := fs.filer.FindEntry(ctx, fullpath)
	if err == filer_pb.ErrNotFound {
		return 0, false, nil
	}
	if err != nil {
		glog.V(3).InfofCtx(ctx, "touchAccessTime %s: lookup: %v", fullpath, err)
		return 0, false, err
	}

	if !policy.ShouldUpdate(entry.Attr, candidate) {
		return 0, false, nil
	}

	entry.Attr.Atime = candidate
	if err := fs.filer.Store.UpdateEntry(ctx, entry); err != nil {
		glog.V(3).InfofCtx(ctx, "touchAccessTime %s: store: %v", fullpath, err)
		return 0, false, err
	}
	return candidate.UnixNano(), true, nil
}

func (fs *FilerServer) bumpAtimeForEntry(ctx context.Context, entry *filer.Entry) {
	if entry == nil || entry.IsDirectory() {
		return
	}
	if atime.IsSystemRead(ctx) {
		return
	}
	policy := fs.atimePolicy()
	if policy.Mode == filer.AtimeModeOff {
		return
	}
	fullpath := entry.FullPath
	now := time.Now()
	detached := context.WithoutCancel(ctx)
	go func() {
		if _, _, err := fs.touchAccessTime(detached, fullpath, now); err != nil {
			glog.V(3).Infof("bumpAtimeForEntry %s: %v", fullpath, err)
		}
	}()
}

func (fs *FilerServer) atimePolicy() *filer.AtimePolicy {
	if fs.option == nil || fs.option.AtimePolicy == nil {
		return &filer.AtimePolicy{Mode: filer.AtimeModeRelatime, RelatimeThreshold: filer.DefaultRelatimeThreshold}
	}
	return fs.option.AtimePolicy
}
