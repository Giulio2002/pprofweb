package pprofweb

import (
	"log"
	"net/http"
)

const rootTemplate = `<!doctype html>
<html>
<head><title>PProf Web Interface</title></head>
<body>
<p>Upload a file to explore it using the <a href="https://github.com/google/pprof">Pprof</a> web interface.</p>

<p>
the source code <a href="https://tuxpa.in/a/pprofweb">over here </a>.
</p>

<p>
code is based off <a href="https://github.com/evanj/pprofweb">this project</a>.
</p>
<p>
when you upload data, you get a unique url. you can share that url with other people!
</p>
<p>
i currently have it set to save forever, but that might not always be true. assume that things can be lost at any time!
</p>

<p>
also assume that things are not private. i am using xid to generate, which i believe is not secure? its also just on a random pvc
</p>

<p>
anyhow, just upload your profile here!
</p>

<p>
go tool pprof [binary] profile.pb.gz
</p>

<p>
works with whatever works in the command line (it just runs go tool pprof in the background anyways)
</p>

<form method="post" action="/upload" enctype="multipart/form-data">
<p>Upload file: <input type="file" name="file"> <input type="submit" value="Upload"></p>
</form>


<p>
u can also use curl! basically it checks if your Accept headers have "http", and if not, then it will not redirect, and will send the id instead
curl -F file=@<filename> https://pprof.tuxpa.in/upload
</p>

<p>
i also made a simple cli
you can download it by doing
</p>
<p>
go install tuxpa.in/a/pprofweb
</p>

</body>
</html>
`

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("rootHandler %s %s", r.Method, r.URL.String())
	if r.Method != http.MethodGet {
		http.Error(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Write([]byte(rootTemplate))
}
