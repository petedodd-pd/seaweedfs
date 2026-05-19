package mount

import (
	"context"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/util/atime"
)

func (wfs *WFS) newAtimeToucher() *atime.Toucher {
	return atime.NewToucher(func(ctx context.Context, dir, name string, atimeNs int64) {
		err := wfs.WithFilerClient(false, func(client filer_pb.SeaweedFilerClient) error {
			_, touchErr := client.TouchAccessTime(ctx, &filer_pb.TouchAccessTimeRequest{
				Directory:     dir,
				Name:          name,
				ClientAtimeNs: atimeNs,
			})
			return touchErr
		})
		if err != nil {
			glog.V(3).Infof("TouchAccessTime %s/%s: %v", dir, name, err)
		}
	})
}

func (wfs *WFS) bumpAtime(ctx context.Context, fullPath string) {
	if wfs == nil || wfs.atimeToucher == nil || fullPath == "" {
		return
	}
	dir, name := splitPath(fullPath)
	if dir == "" || name == "" {
		return
	}
	wfs.atimeToucher.Bump(ctx, dir, name)
}

func splitPath(fullPath string) (string, string) {
	for i := len(fullPath) - 1; i >= 0; i-- {
		if fullPath[i] == '/' {
			if i == 0 {
				return "/", fullPath[1:]
			}
			return fullPath[:i], fullPath[i+1:]
		}
	}
	return "", fullPath
}
