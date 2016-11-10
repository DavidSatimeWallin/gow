APP_NAME=gow ; rm -rf $APP_NAME ; go build -ldflags="-w -s" -a -installsuffix cgo -o tmpfile ./ ; strip -x tmpfile -o $APP_NAME ; rm -rf tmpfile ; upx -9 $APP_NAME
