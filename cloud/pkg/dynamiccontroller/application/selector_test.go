package application

/*
func TestLabelFieldSelector_MarshalJSON(t *testing.T) {
	type fields struct {
		Label labels.Selector
		Field k8sfields.Selector
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "Test UnMarshalJSON",
			fields:  fields{Label: labels.SelectorFromSet(map[string]string{"kubeedge.io/label-foo": "true"}), Field: k8sfields.SelectorFromSet(map[string]string{"metadata.namespace": "default"})},
			want:    []byte(`"kubeedge.io/label-foo=true;metadata.namespace=default"`),
			wantErr: false,
		},
		{
			name:    "Test UnMarshalJSON",
			fields:  fields{},
			want:    []byte(`";"`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lf := &LabelFieldSelector{
				Label: tt.fields.Label,
				Field: tt.fields.Field,
			}
			got, err := json.Marshal(lf)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestLabelFieldSelector_UnmarshalJSON(t *testing.T) {
	type fields struct {
		Label labels.Selector
		Field k8sfields.Selector
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "Test UnMarshalJSON",
			fields:  fields{Label: labels.SelectorFromSet(map[string]string{"kubeedge.io/label-foo": "true"}), Field: k8sfields.SelectorFromSet(map[string]string{"metadata.namespace": "default"})},
			args:    args{[]byte{}},
			wantErr: false,
		},
		{
			name:    "Test UnMarshalJSON",
			fields:  fields{Label: labels.SelectorFromSet(map[string]string{"kubeedge.io/label-foo": "true"}), Field: k8sfields.SelectorFromSet(map[string]string{})},
			args:    args{[]byte{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := &LabelFieldSelector{
				Label: tt.fields.Label,
				Field: tt.fields.Field,
			}
			tt.args.b, _ = json.Marshal(want)
			got := new(LabelFieldSelector)
			if err := json.Unmarshal(tt.args.b,got); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("UnmarshalJSON() got = {%v}, want = {%v}", got, want)
			}
			klog.Infof("want: %+v",*want)
		})
	}
}
*/
