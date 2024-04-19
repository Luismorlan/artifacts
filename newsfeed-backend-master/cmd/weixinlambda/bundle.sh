GOOS=linux GOARCH=amd64 go build main.go
zip lambda.zip main
aws lambda update-function-code \
  --region us-west-2 \
  --function-name arn:aws:lambda:us-west-2:213288384225:function:weixin \
  --zip-file fileb://lambda.zip \
  --profile rnr
rm lambda.zip main
