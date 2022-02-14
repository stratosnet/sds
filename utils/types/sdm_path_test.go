package types

import (
	"reflect"
	"testing"
)

func TestDataMashIdFromString(t *testing.T) {
	type args struct {
		idString string
	}
	tests := []struct {
		name    string
		args    args
		want    *DataMashId
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "1", args: args{"sdm://abasdfa/asdfasdf"}, want: &DataMashId{
			Owner: "abasdfa",
			Hash:  "asdfasdf",
		}, wantErr: false},
		{name: "2", args: args{"sdm://abasdfa//asdfasdf"}, want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DataMashIdFromString(tt.args.idString)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataMashIdFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataMashIdFromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataMashId_String(t *testing.T) {
	type fields struct {
		Owner string
		Hash  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
		{name: "1", fields: fields{
			Owner: "aqwerqwerq",
			Hash:  "sfdgsdfasidfj",
		}, want: "sdm://aqwerqwerq/sfdgsdfasidfj"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DataMashId{
				Owner: tt.fields.Owner,
				Hash:  tt.fields.Hash,
			}
			if got := d.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
