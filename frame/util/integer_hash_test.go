package util

import "testing"

func TestHash32(t *testing.T) {
	type args struct {
		x uint32
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{"", args{123}, 123},
		{"", args{435123}, 435123},
		{"", args{935947434}, 935947434},
		{"", args{3548783459}, 3548783459},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnHash32(Hash32(tt.args.x)); got != tt.want {
				t.Errorf("Hash32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash64(t *testing.T) {
	type args struct {
		x uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"", args{123}, 123},
		{"", args{435123}, 435123},
		{"", args{935947434}, 935947434},
		{"", args{3548783459}, 3548783459},
		{"", args{354878345912345}, 354878345912345},
		{"", args{35487834591234522}, 35487834591234522},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnHash64(Hash64(tt.args.x)); got != tt.want {
				t.Errorf("Hash64() = %v, want %v", got, tt.want)
			}
		})
	}
}
