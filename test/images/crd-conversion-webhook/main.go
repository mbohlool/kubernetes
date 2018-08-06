/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

// Config contains the server (the webhook) cert and key.
type Config struct {
	CertFile string
	KeyFile  string
}

func (c *Config) addFlags() {
	flag.StringVar(&c.CertFile, "tls-cert-file", c.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	flag.StringVar(&c.KeyFile, "tls-private-key-file", c.KeyFile, ""+
		"File containing the default x509 private key matching --tls-cert-file.")
}

func convertCRD(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, error) {
	glog.V(2).Info("converting crd")

	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, fmt.Errorf("conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	switch Object.GetAPIVersion() {
	case "stable.example.com/v1":
		switch toVersion {
		case "stable.example.com/v2":
			hostPort, ok := convertedObject.Object["hostPort"]
			if !ok {
				return nil, fmt.Errorf("expected hostPort in stable.example.com/v1")
			}
			delete(convertedObject.Object, "hostPort")
			parts := strings.Split(hostPort.(string), ":")
			convertedObject.Object["host"] = parts[0]
			convertedObject.Object["port"] = parts[1]
		default:
			return nil, fmt.Errorf("unexpected conversion version %s", toVersion)
		}
	case "stable.example.com/v2":
		switch toVersion {
		case "stable.example.com/v1":
			host, ok := convertedObject.Object["host"]
			if !ok {
				return nil, fmt.Errorf("expected host in stable.example.com/v2")
			}
			port, ok := convertedObject.Object["port"]
			if !ok {
				return nil, fmt.Errorf("expected port in stable.example.com/v2")
			}
			delete(convertedObject.Object, "host")
			delete(convertedObject.Object, "port")
			convertedObject.Object["hostPort"] = fmt.Sprintf("%s:%s", host, port)
		default:
			return nil, fmt.Errorf("unexpected conversion version %s", toVersion)
		}

	default:
		return nil, fmt.Errorf("unexpected conversion version %s", fromVersion)
	}

	return convertedObject, nil
}

type convertFunc func(Object *unstructured.Unstructured, version string) (*unstructured.Unstructured, error)

func serve(w http.ResponseWriter, r *http.Request, convert convertFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	glog.V(2).Info(fmt.Sprintf("handling request: %v", body))
	convertReview := v1beta1.ConversionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &convertReview); err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(convertReview.Request.Object.Raw); err != nil {
			glog.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var convertedCR *unstructured.Unstructured
		var err error
		if convertReview.Request.IsList {
			listCR, err := cr.ToList()
			if err != nil {
				glog.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			convertedList := listCR.DeepCopy()
			for i := 0; i < len(convertedList.Items); i++  {
				item, err := convert(&convertedList.Items[i], convertReview.Request.APIVersion)
				if err != nil {
					glog.Error(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				item.SetAPIVersion(convertReview.Request.APIVersion)
				convertedList.Items[i] = *item
			}
			convertedCR = &unstructured.Unstructured{}
			convertedCR.SetUnstructuredContent(convertedList.UnstructuredContent())
		} else {
			convertedCR, err = convert(&cr, convertReview.Request.APIVersion)
			if err != nil {
				glog.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		convertedCR.SetAPIVersion(convertReview.Request.APIVersion)
		convertReview.Response.ConvertedObject = runtime.RawExtension{
			Object: convertedCR,
		}
	}
	glog.V(2).Info(fmt.Sprintf("sending response: %v", convertReview.Response))

	convertReview.Response.UID = convertReview.Request.UID
	// reset the request, it is not needed in a response.
	convertReview.Request = &v1beta1.ConversionRequest{}

	resp, err := json.Marshal(convertReview)
	if err != nil {
		glog.Error(err)
	}
	if _, err := w.Write(resp); err != nil {
		glog.Error(err)
	}
}

func serveConvert(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertCRD)
}

func main() {
	var config Config
	config.addFlags()
	flag.Parse()

	http.HandleFunc("/convert", serveConvert)
	clientset := getClient()
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: configTLS(config, clientset),
	}
	server.ListenAndServeTLS("", "")
}
