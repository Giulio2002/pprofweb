# PProf Web UI

THIS IS A FORK

[Go pprof profiler web UI](https://github.com/google/pprof).

You can upload pprof files then view them without installing anything.


you can also share these profiles with other people


Try it: https://pprof.tuxpa.in


## Run Locally

docker build . --tag=pprofweb
docker run --rm -ti -p 7443:7443 pprofweb

Open http://localhost:7443/


## Check that the container works

docker run --rm -ti --entrypoint=dot pprofweb
