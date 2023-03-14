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
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/maven"
)

type jibTask struct {
	c     client.Client
	build *v1.Build
	task  *v1.JibTask
}

var _ Task = &jibTask{}

func (t *jibTask) Do(ctx context.Context) v1.BuildStatus {
	status := v1.BuildStatus{}

	baseImage := t.build.Status.BaseImage
	if baseImage == "" {
		baseImage = t.task.BaseImage
		status.BaseImage = baseImage
	}

	mavenConfig := t.task.Maven
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-integration", defaults.Version)
	p.DependencyManagement = &maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)}
	p.Dependencies = make([]maven.Dependency, 0)
	p.Build = &maven.Build{Plugins: make([]maven.Plugin, 0)}
	// TODO add the plugin, it is enougth

	// FIXME : remove this code later
	mavenJson, _ := json.Marshal(mavenConfig)
	log.Errorf(errors.New("Jib error from gfournie"), "Maven: %s", string(mavenJson))

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

	pullInsecure := t.task.Registry.Insecure // incremental build case

	log.Errorf(errors.New("Jib error from gfournie"), "Jib error from gfournie > listing folder")
	errRead := listFolder(contextDir)
	log.Errorf(errRead, "Jib error from gfournie > displaying listing file")
	displayFile(contextDir + "/../maven/pom.xml")
	log.Errorf(errRead, "Jib error from gfournie > finished file")

	log.Debugf("Registry address: %s", t.task.Registry.Address)
	log.Errorf(errors.New("Jib error from gfournie"), "Registry address: %s", t.task.Registry.Address)
	log.Debugf("Base image: %s", baseImage)
	log.Errorf(errors.New("Jib error from gfournie"), "Base image: %s", baseImage)

	if !strings.HasPrefix(baseImage, t.task.Registry.Address) {
		if pullInsecure {
			log.Info("Assuming secure pull because the registry for the base image and the main registry are different")
			pullInsecure = false
		}
	}

	registryConfigDir := ""
	if t.task.Registry.Secret != "" {
		registryConfigDir, err = mountSecret(ctx, t.c, t.build.Namespace, t.task.Registry.Secret)
		if err != nil {
			return status.Failed(err)
		}
	}

	if registryConfigDir != "" {
		if err := os.RemoveAll(registryConfigDir); err != nil {
			return status.Failed(err)
		}
	}

	mc := maven.NewContext(contextDir)

	//mc.GlobalSettings = mavenConfig.Settings
	//mc.UserSettings = mavenConfig.User
	//mc.SettingsSecurity = mavenConfig.SettingsSecurity
	mc.LocalRepository = mavenConfig.LocalRepository
	mc.AdditionalArguments = mavenConfig.CLIOptions

	//if ctx.Maven.TrustStoreName != "" {
	//	mc.ExtraMavenOpts = append(mc.ExtraMavenOpts,
	//		"-Djavax.net.ssl.trustStore="+filepath.Join(ctx.Path, ctx.Maven.TrustStoreName),
	//		"-Djavax.net.ssl.trustStorePassword="+ctx.Maven.TrustStorePass,
	//	)
	//}

	// Run the Maven goal
	//if err := maven.Project.Command(mc).Do(ctx); err != nil {
	//	return errors.Wrap(err, "failure while generating image")
	//}

	log.Errorf(errors.New("Jib error from gfournie"), "Doing nothing in Jib sorry")
	return status
}

func displayFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf(err, "Shit happened")
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Errorf(err, "Shit happened when closing file")
		}
	}()

	b, err := ioutil.ReadAll(file)
	//fmt.Print(b)
	log.Errorf(errors.New("Jib error from gfournie > this is the file"), string(b[:]))
}

func listFolder(filePath string) error {
	errRead := filepath.Walk(filePath, func(path string, info os.FileInfo, errRead error) error {
		if errRead != nil {
			log.Errorf(errRead, "Jib error from gfournie")
			return errRead
		}
		msg := "dir: " + strconv.FormatBool(info.IsDir()) + ": name: " + path + "\n"
		log.Errorf(errors.New("Jib error from gfournie"), msg)

		return nil
	})
	if errRead != nil {
		log.Errorf(errRead, "Jib error from gfournie")
	}
	return errRead
}
