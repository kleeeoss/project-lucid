module lucid-ci/platform

go 1.23

require (
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.9
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.7.2
	lucid-ci/engine v0.0.0
)

replace lucid-ci/engine => ../engine
