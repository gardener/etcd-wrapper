package app

import "net/http"

func (a *Application) readinessHandler(w http.ResponseWriter, r *http.Request) {

}

func (a *Application) SetupReadinessProbe() {
	//the caller should call this in a go-routine as this method will setup a HTTP router and call ListenAndServe which blocks.
	//If the http server errors out then you will have to panic and that should cause the container to exit and then be restarted by kubelet.
}
