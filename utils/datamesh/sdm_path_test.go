package datamesh

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
		{name: "1", args: args{"sdm://st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg/v05ahm52iirbq55177uii2bmbsmmcemnjtm740s8"}, want: &DataMashId{
			Owner: "st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg",
			Hash:  "v05ahm52iirbq55177uii2bmbsmmcemnjtm740s8",
		}, wantErr: false},
		{name: "2", args: args{"sdm://st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg//v05ahm52iirbq55177uii2bmbsmmcemnjtm740s8"}, want: nil, wantErr: true},
		{name: "3", args: args{"sdm://st1jn9skjsnxv26mekd8eu8a8aquh30m4mwgahg/v05ahm52iirbq55177uii2bmbsmmcemnjtm740s8"}, want: nil, wantErr: true},
		{name: "4", args: args{"sdm://st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg/v05ahm52iirbq55177u2bmbsmmcemnjtm740s8"}, want: nil, wantErr: true},
		{name: "5", args: args{"sdm://st1jn9skjsnxv26mekd8eu8a8aq3h34v0m4mwgahg/v05ahm52iirbq55177uii2bmbsmmcemnjtm740s8"}, want: nil, wantErr: true},
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
			Owner: "st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg",
			Hash:  "v05ahm52iirbq55177uii2bmbsmmcemnjtm740s8",
		}, want: "sdm://st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg/v05ahm52iirbq55177uii2bmbsmmcemnjtm740s8"},
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
