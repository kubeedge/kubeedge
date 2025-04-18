package handlerfactory

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// PassThrough handle with the pass through request
func (f *Factory) PassThrough() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		options := metav1.GetOptions{}
		result, err := f.storage.PassThrough(req.Context(), &options)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(result); err != nil {
			// TODO: handle error
			klog.Error(err)
		}
	})
	return h
}
