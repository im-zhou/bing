# !/bin/sh
workpath=$(dirname $0)
cd ${workpath}
nohup supervisor node bin/www >app.log &
