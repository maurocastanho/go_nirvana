#!/bin/bash
fname=xls2xml_$(/bin/date +%Y%m%d).zip
rm ${fname}
GOPATH=/home/mauro/go/windows GOOS=windows GOARCH=amd64 go build && \
zip ${fname} xls2xml.exe config_box.json config_net.json config_oi_ott.json
