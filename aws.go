package duat

// Utility functions for interacting with AWS

import (
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/jjeffery/kv"     // Forked copy of https://github.com/jjeffery/kv
	"github.com/go-stack/stack" // Forked copy of https://github.com/go-stack/stack
)

func GetECRToken() (token string, err kv.Error) {
	svc := ecr.New(session.New())
	input := &ecr.GetAuthorizationTokenInput{}

	result, errGo := svc.GetAuthorizationToken(input)
	if errGo != nil {
		if aerr, ok := errGo.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return "", kv.NewError(ecr.ErrCodeServerException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeInvalidParameterException:
				return "", kv.NewError(ecr.ErrCodeInvalidParameterException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			default:
				return "", kv.Wrap(aerr).With("stack", stack.Trace().TrimRuntime())
			}
		} else {
			return "", kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}
	if len(result.AuthorizationData) != 1 {
		return "", kv.NewError("aws auth data is in an unknown format").With("stack", stack.Trace().TrimRuntime())
	}

	return *result.AuthorizationData[0].AuthorizationToken, nil
}

func CreateECRRepo(repo string) (err kv.Error) {
	svc := ecr.New(session.New())
	input := &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repo),
	}

	if _, errGo := svc.CreateRepository(input); errGo != nil {
		if aerr, ok := errGo.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return kv.NewError(ecr.ErrCodeServerException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeInvalidParameterException:
				return kv.NewError(ecr.ErrCodeInvalidParameterException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeRepositoryAlreadyExistsException:
				return kv.Wrap(aerr).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeLimitExceededException:
				return kv.NewError(ecr.ErrCodeLimitExceededException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			default:
				return kv.Wrap(aerr).With("stack", stack.Trace().TrimRuntime())
			}
		} else {
			return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}
	return nil
}

func GetECRDefaultURL() (urlOut *url.URL, err kv.Error) {

	svc := ecr.New(session.New())
	input := &ecr.DescribeRepositoriesInput{MaxResults: aws.Int64(1)}

	result, errGo := svc.DescribeRepositories(input)
	if errGo != nil {
		if aerr, ok := errGo.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return nil, kv.NewError(ecr.ErrCodeServerException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeInvalidParameterException:
				return nil, kv.NewError(ecr.ErrCodeInvalidParameterException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeRepositoryNotFoundException:
				return nil, kv.NewError(ecr.ErrCodeRepositoryNotFoundException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			default:
				return nil, kv.Wrap(aerr).With("stack", stack.Trace().TrimRuntime())
			}
		} else {
			return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}
	for _, repo := range result.Repositories {
		if repo.RepositoryUri == nil {
			continue
		}
		urlOut, errGo = url.Parse("https://" + *repo.RepositoryUri)
		if errGo != nil {
			return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		return urlOut, nil
	}

	return nil, nil
}
