package spanalyzer

import (
	"fmt"

	googlesql "github.com/goccy/go-googlesql"
	"google.golang.org/protobuf/proto"
)

func buildGoogleSQLDescriptorPool(descriptors *ProtoDescriptorSet) (*googlesql.DescriptorPool, error) {
	if descriptors == nil || descriptors.fileSet == nil {
		return nil, nil
	}
	pool, err := googlesql.NewDescriptorPool()
	if err != nil {
		return nil, err
	}
	remaining := make(map[int]struct{}, len(descriptors.fileSet.File))
	for i := range descriptors.fileSet.File {
		remaining[i] = struct{}{}
	}
	for len(remaining) > 0 {
		progressed := false
		var lastErr error
		for i := range remaining {
			file := descriptors.fileSet.File[i]
			fdp, err := googlesql.NewFileDescriptorProto()
			if err != nil {
				return nil, err
			}
			wire, err := proto.Marshal(file)
			if err != nil {
				return nil, err
			}
			ok, err := fdp.ParseFromString(string(wire))
			if err != nil {
				return nil, fmt.Errorf("parse file descriptor %q: %w", file.GetName(), err)
			}
			if !ok {
				return nil, fmt.Errorf("parse file descriptor %q failed", file.GetName())
			}
			desc, err := pool.BuildFile(fdp)
			if err != nil {
				lastErr = fmt.Errorf("build file descriptor %q: %w", file.GetName(), err)
				continue
			}
			if desc == nil {
				lastErr = fmt.Errorf("build file descriptor %q returned nil", file.GetName())
				continue
			}
			delete(remaining, i)
			progressed = true
		}
		if !progressed {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("build proto descriptor pool made no progress")
		}
	}
	return pool, nil
}
