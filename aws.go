package duat

// Utility functions for interacting with AWS

import (
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

func GetECRToken() (token string, err errors.Error) {
	svc := ecr.New(session.New())
	input := &ecr.GetAuthorizationTokenInput{}

	result, errGo := svc.GetAuthorizationToken(input)
	if errGo != nil {
		if aerr, ok := errGo.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return "", errors.New(ecr.ErrCodeServerException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeInvalidParameterException:
				return "", errors.New(ecr.ErrCodeInvalidParameterException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			default:
				return "", errors.Wrap(aerr).With("stack", stack.Trace().TrimRuntime())
			}
		} else {
			return "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}
	if len(result.AuthorizationData) != 1 {
		return "", errors.New("aws auth data is in an unknown format").With("stack", stack.Trace().TrimRuntime())
	}

	return *result.AuthorizationData[0].AuthorizationToken, nil
}

func CreateECRRepo(repo string) (err errors.Error) {
	svc := ecr.New(session.New())
	input := &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repo),
	}

	if _, errGo := svc.CreateRepository(input); errGo != nil {
		if aerr, ok := errGo.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return errors.New(ecr.ErrCodeServerException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeInvalidParameterException:
				return errors.New(ecr.ErrCodeInvalidParameterException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeRepositoryAlreadyExistsException:
				return errors.Wrap(aerr).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeLimitExceededException:
				return errors.New(ecr.ErrCodeLimitExceededException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			default:
				return errors.Wrap(aerr).With("stack", stack.Trace().TrimRuntime())
			}
		} else {
			return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}
	return nil
}

func GetECRDefaultURL() (urlOut *url.URL, err errors.Error) {

	svc := ecr.New(session.New())
	input := &ecr.DescribeRepositoriesInput{MaxResults: aws.Int64(1)}

	result, errGo := svc.DescribeRepositories(input)
	if errGo != nil {
		if aerr, ok := errGo.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return nil, errors.New(ecr.ErrCodeServerException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeInvalidParameterException:
				return nil, errors.New(ecr.ErrCodeInvalidParameterException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			case ecr.ErrCodeRepositoryNotFoundException:
				return nil, errors.New(ecr.ErrCodeRepositoryNotFoundException).With("cause", aerr.Error()).With("stack", stack.Trace().TrimRuntime())
			default:
				return nil, errors.Wrap(aerr).With("stack", stack.Trace().TrimRuntime())
			}
		} else {
			return nil, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}
	for _, repo := range result.Repositories {
		if repo.RepositoryUri == nil {
			continue
		}
		urlOut, errGo = url.Parse("https://" + *repo.RepositoryUri)
		if errGo != nil {
			return nil, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		return urlOut, nil
	}

	return nil, nil
}
