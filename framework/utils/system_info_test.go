package utils

import (
	"reflect"
	"testing"

	"github.com/shirou/gopsutil/disk"
)

func TestGetDiskUsage(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *disk.UsageStat
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDiskUsage(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDiskUsage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDiskUsage() got = %v, want %v", got, tt.want)
			}
		})
	}

}
