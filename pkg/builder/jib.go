/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

type jibTask struct {
	c     client.Client
	build *v1.Build
	task  *v1.JibTask
}

var _ Task = &jibTask{}

var (
	logger = log.WithName("jib")

	loggerInfo  = func(s string) string { logger.Info(s); return "" }
	loggerError = func(s string) string { logger.Error(nil, s); return "" }
)

func (t *jibTask) Do(ctx context.Context) v1.BuildStatus {
	status := v1.BuildStatus{}
	log.Infof("Doing things with jib")

	baseImage := t.build.Status.BaseImage
	if baseImage == "" {
		baseImage = t.task.BaseImage
		status.BaseImage = baseImage
	}

	contextDir := t.task.ContextDir
	if contextDir == "" {
		// Use the working directory.
		// This is useful when the task is executed in-container,
		// so that its WorkingDir can be used to share state and
		// coordinate with other tasks.
		pwd, err := os.Getwd()
		if err != nil {
			return status.Failed(err)
		}
		contextDir = filepath.Join(pwd, ContextDir)
	}

	exists, err := util.DirectoryExists(contextDir)
	if err != nil {
		return status.Failed(err)
	}
	empty, err := util.DirectoryEmpty(contextDir)
	if err != nil {
		return status.Failed(err)
	}
	if !exists || empty {
		// this can only indicate that there are no more resources to add to the base image,
		// because transitive resolution is the same even if spec differs.
		log.Infof("No new image to build, reusing existing image %s", baseImage)
		status.Image = baseImage
		return status
	}
	mavenDir := strings.ReplaceAll(contextDir, "context", "maven")

	pullInsecure := t.task.Registry.Insecure // incremental build case

	if !strings.HasPrefix(baseImage, t.task.Registry.Address) {
		if pullInsecure {
			log.Info("Assuming secure pull because the registry for the base image and the main registry are different")
			pullInsecure = false
		}
	}

	registryConfigDir := ""
	if t.task.Registry.Secret != "" {
		registryConfigDir, err = MountSecret(ctx, t.c, t.build.Namespace, t.task.Registry.Secret)
		if err != nil {
			return status.Failed(err)
		}
	}

	if registryConfigDir != "" {
		if err := os.RemoveAll(registryConfigDir); err != nil {
			return status.Failed(err)
		}
	}

	// Then do temp things because

	log.Infof("Registry address: %s", t.task.Registry.Address)
	log.Infof("Base image: %s", baseImage)

	log.Info("Jib start from gfournie > displaying file MAVEN_CONTEXT")
	mavenCommand, err := readFile(mavenDir + "/MAVEN_CONTEXT")
	if err != nil {
		return status.Failed(err)
	}
	log.Info("Jib end from gfournie > finished file MAVEN_CONTEXT")

	log.Info("SLEEPING for 10")
	time.Sleep(10 * time.Second)
	log.Info("WAKEY WAKEY")

	// trying
	mavenCommandStr := string(mavenCommand)
	mavenCommandStr = strings.ReplaceAll(mavenCommandStr, "./mvnw", "")
	mavenCommandStr = strings.Trim(mavenCommandStr, " ")
	mavenArgs := strings.Split(mavenCommandStr, " ")
	log.Info("|" + mavenCommandStr + "|")
	log.Info(strings.Join(mavenArgs, "|"))

	log.Info("SLEEPING for 10")
	time.Sleep(10 * time.Second)
	log.Info("WAKEY WAKEY")

	mvnCmd := "./mvnw"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}
	cmd := exec.CommandContext(ctx, mvnCmd, mavenArgs...)
	cmd.Dir = mavenDir

	myerror := util.RunAndLog(ctx, cmd, loggerError, loggerInfo)
	if myerror != nil {
		log.Errorf(myerror, "jib integration image containerization did not run successfully")
		return status.Failed(myerror)
	} else {
		log.Info("everything went so well T_T")
	}

	return status
}

func readFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf(err, "Shit happened")
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Errorf(err, "Shit happened when closing file")
		}
	}()

	return ioutil.ReadAll(file)

}
