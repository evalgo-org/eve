# eve

## run sample flow-service
```
go run main.go --config example.config.yaml
```

## generate JWT token
```
JWT_TOKEN=$(curl -XPOST -d '{"user_id": "1234567890"}' -H "Content-Type: application/json" -s http://localhost:8080/auth/token | jq -r .token)
```

## publish a process_id
```
curl -XPOST -d '{"process_id":"1234567890"}' -H "Content-Type: application/json" -H "Authorization: Bearer ${JWT_TOKEN}" -s http://localhost:8080/api/publish | jq .
```

## get a process list
```
 curl -XGET -H "Authorization: Bearer ${JWT_TOKEN}" -s http://localhost:8080/api/processes | jq .
```

