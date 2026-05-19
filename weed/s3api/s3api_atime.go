package s3api

import (
	"context"
	"path"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/util/atime"
)

func (s3a *S3ApiServer) newAtimeToucher() *atime.Toucher {
	return atime.NewToucher(func(ctx context.Context, dir, name string, atimeNs int64) {
		err := s3a.WithFilerClient(false, func(client filer_pb.SeaweedFilerClient) error {
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

func (s3a *S3ApiServer) bumpAtime(ctx context.Context, bucket, object string) {
	if s3a == nil || s3a.atimeToucher == nil {
		return
	}
	if bucket == "" || object == "" {
		return
	}
	fullPath := s3a.toFilerPath(bucket, object)
	dir, name := path.Split(fullPath)
	dir = trimTrailingSlash(dir)
	if dir == "" || name == "" {
		return
	}
	s3a.atimeToucher.Bump(ctx, dir, name)
}

func trimTrailingSlash(s string) string {
	if len(s) > 1 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
