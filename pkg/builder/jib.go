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

	// TODO manage registry auth conf
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

	// Then do poc actions for jib
	log.Infof("Registry address: %s", t.task.Registry.Address)
	log.Infof("Registry secret: %s", t.task.Registry.Secret)
	log.Infof("Base image: %s", baseImage)

	errCreateJibConfig := createJibConfig(ctx, contextDir)
	if errCreateJibConfig != nil {
		return status.Failed(errCreateJibConfig)
	}
	// TODO temp
	jibConfig, errConfig := readFile(filepath.Join(contextDir, "config.json"))
	if errConfig != nil {
		return status.Failed(errConfig)
	}
	log.Info(string(jibConfig))
	// TODO temp

	jibCmd := "/opt/jib/bin/jib"
	jibArgs := []string{"build",
		"--verbosity=info",
		"--target=" + t.task.Image,
		"--allow-insecure-registries",
		"--build-file=" + filepath.Join(contextDir, "jibclibuild.yaml"),
		"--image-metadata-out=" + filepath.Join(contextDir, "jibimage.json")}

	cmd := exec.CommandContext(ctx, jibCmd, jibArgs...)

	cmd.Dir = contextDir

	env := os.Environ()
	env = append(env, "XDG_CONFIG_HOME="+contextDir)
	env = append(env, "HOME="+contextDir)
	cmd.Env = env

	myerror := util.RunAndLog(ctx, cmd, loggerInfo, loggerError)
	if myerror != nil {
		log.Errorf(myerror, "jib integration image containerization did not run successfully")
		return status.Failed(myerror)
	} else {
		log.Info("jib integration image containerization did run successfully")
		status.Image = t.task.Image

		// retrieve image digest
		jibOutput, errDigest := readFile(filepath.Join(contextDir, "jibimage.json"))
		if errDigest != nil {
			return status.Failed(errDigest)
		}
		log.Info(string(jibOutput))
		// TODO  extract digest
		//status.Digest = string("mavenDigest")
	}

	if registryConfigDir != "" {
		if err := os.RemoveAll(registryConfigDir); err != nil {
			return status.Failed(err)
		}
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

func createJibConfig(ctx context.Context, jibContextDir string) error {
	// #nosec G202
	jibConfig := []byte(`{"disableUpdateCheck": true, "registryMirrors": []}`)

	log.Info(filepath.Join(jibContextDir, "config.json"))
	err := os.WriteFile(filepath.Join(jibContextDir, "config.json"), jibConfig, 0o755)
	if err != nil {
		return err
	}

	return nil

}
