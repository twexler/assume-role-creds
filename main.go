package main

import (
	"flag"
	"log"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-ini/ini"
	homedir "github.com/mitchellh/go-homedir"
)

type credential struct {
	AWSAccessKeyID    string `ini:"aws_access_key_id"`
	AWSSecretAcessKey string `ini:"aws_secret_access_key"`
	AWSSessionToken   string `ini:"aws_session_token"`
}

var (
	childProfileName  = flag.String("childProfile", "", "the name of the profile to create/modify")
	credsPath         = flag.String("credentials", "", "the path to the credentials file")
	roleARN           = flag.String("roleArn", "", "the ARN of the role to assume")
	parentProfileName = flag.String("parentProfile", "default", "the name of the profile to assume the role from")
)

func main() {
	flag.Parse()
	if *childProfileName == "" {
		log.Fatalf("No child profile name specified")
	}
	if *roleARN == "" {
		log.Fatalln("No role ARN specified")
	}
	if *credsPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			panic(err)
		}
		p := path.Join(home, ".aws", "credentials")
		credsPath = &p
	}
	config := aws.NewConfig().WithCredentials(credentials.NewSharedCredentials(*credsPath, *parentProfileName))
	config = config.WithMaxRetries(3)
	sess := session.Must(session.NewSession(config))
	svc := sts.New(sess, config)
	resp, err := svc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         roleARN,
		RoleSessionName: childProfileName,
	})
	if err != nil {
		log.Fatalf("Unable to assume role: %s\n", err.Error())
	}
	iniFile, err := ini.Load(*credsPath)
	if err != nil {
		log.Fatalf("Unable to load credentials file for writing: %s\n", err.Error())
	}
	cred := &credential{
		AWSAccessKeyID:    *resp.Credentials.AccessKeyId,
		AWSSecretAcessKey: *resp.Credentials.SecretAccessKey,
		AWSSessionToken:   *resp.Credentials.SessionToken,
	}
	if sec, _ := iniFile.GetSection(*childProfileName); sec != nil {
		// existing profile
		if mapErr := sec.ReflectFrom(cred); mapErr != nil {
			log.Fatalf("Unable to write credentials: %s\n", err.Error())
		}
	} else {
		sec, newSecErr := iniFile.NewSection(*childProfileName)
		if newSecErr != nil {
			log.Fatalf("Unable to create new section: %s\n", newSecErr.Error())
		}
		if mapErr := sec.ReflectFrom(cred); mapErr != nil {
			log.Fatalf("Unable to map credentials: %s\n", mapErr.Error())
		}
	}
	if saveErr := iniFile.SaveTo(*credsPath); err != nil {
		log.Fatalf("Unable to write credentials file: %s\n", saveErr.Error())
	}
	log.Println("Updated tokens")
}
