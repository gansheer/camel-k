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

package catalog

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/client"
	platformutil "github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	spectrum "github.com/container-tools/spectrum/pkg/builder"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// NewInitializeAction returns a action that initializes the catalog configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(catalog *v1.CamelCatalog) bool {
	return catalog.Status.Phase == v1.CamelCatalogPhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	platform, err := platformutil.GetOrFindLocal(ctx, action.client, catalog.Namespace)

	if err != nil {
		return catalog, err
	}

	if platform.Status.Phase != v1.IntegrationPlatformPhaseReady {
		// Wait the platform to be ready
		return catalog, nil
	}

	if platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyS2I {
		return initializeS2I(ctx, action.client, platform, catalog)
		// return catalog, errors.New("S2I is not yet available - BUT SOON")
	} else if platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategySpectrum {
		// default to Spectrum
		// Make basic options for building image in the registry
		options, err := makeSpectrumOptions(ctx, action.client, platform.Namespace, platform.Status.Build.Registry)
		if err != nil {
			return catalog, err
		}
		return initializeSpectrum(options, platform, catalog)
	}
	// TODO fix
	return catalog, errors.New("Invalid publish strategy " + string(platform.Status.Build.PublishStrategy))
}

func initializeS2I(ctx context.Context, c client.Client, ip *v1.IntegrationPlatform, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	Log.Infof("S2I container Registry is %s", ip.Status.Build.Registry.Address)
	Log.Infof("S2I Tooling base image is %s", catalog.Spec.GetQuarkusToolingImage())
	target := catalog.DeepCopy()

	// No registry in s2i
	imageName := fmt.Sprintf(
		"camel-k-runtime-%s-builder",
		catalog.Spec.Runtime.Provider,
	)
	imageTag := strings.ToLower(catalog.Spec.Runtime.Version)

	/*imageNameFull := fmt.Sprintf(
		"%s/camel-k-runtime-%s-builder:%s",
		ip.Status.Build.Registry.Address,
		catalog.Spec.Runtime.Provider,
		strings.ToLower(catalog.Spec.Runtime.Version),
	)*/
	/*
		dockerfile := string([]byte(`
			FROM ` + catalog.Spec.GetQuarkusToolingImage() + `
			ADD /usr/local/bin/kamel /usr/local/bin/kamel
			ADD /usr/share/maven/mvnw/ /usr/share/maven/mvnw/
			ENV MAVEN_USER_HOME="/usr/share/maven"
			ENV MAVEN_HOME="/usr/share/maven"
			USER 0
			RUN mkdir --chown=nonroot:root /builder
			USER 1000
		`))
	*/

	// Dockfile
	dockerfile := string([]byte(`
		FROM ` + catalog.Spec.GetQuarkusToolingImage() + `
		USER 1000
		ADD /usr/local/bin/kamel /usr/local/bin/kamel
		ADD /usr/share/maven/mvnw/ /usr/share/maven/mvnw/
	`))

	// BuildConfig
	bc := &buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: buildv1.GroupVersion.String(),
			Kind:       "BuildConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageName,
			Namespace: ip.Namespace,
			Labels: map[string]string{
				kubernetes.CamelCreatorLabelKind:      v1.CamelCatalogKind,
				kubernetes.CamelCreatorLabelName:      catalog.Name,
				kubernetes.CamelCreatorLabelNamespace: catalog.Namespace,
				kubernetes.CamelCreatorLabelVersion:   catalog.ResourceVersion,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: catalog.APIVersion,
					Kind:       catalog.Kind,
					Name:       catalog.Name,
					UID:        catalog.UID,
				},
			},
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type:       buildv1.BuildSourceBinary,
					Dockerfile: &dockerfile,
				},
				Strategy: buildv1.BuildStrategy{
					DockerStrategy: &buildv1.DockerBuildStrategy{NoCache: true /*, ForcePull: true*/},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: imageName + ":" + imageTag,
					},
				},
			},
		},
	}

	// ImageStream
	is := &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.GroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      bc.Name,
			Namespace: bc.Namespace,
			Labels: map[string]string{
				kubernetes.CamelCreatorLabelKind:      v1.CamelCatalogKind,
				kubernetes.CamelCreatorLabelName:      catalog.Name,
				kubernetes.CamelCreatorLabelNamespace: catalog.Namespace,
				kubernetes.CamelCreatorLabelVersion:   catalog.ResourceVersion,
				"camel.apache.org/runtime.version":    catalog.Spec.Runtime.Version,
				"camel.apache.org/runtime.provider":   string(catalog.Spec.Runtime.Provider),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: catalog.APIVersion,
					Kind:       catalog.Kind,
					Name:       catalog.Name,
					UID:        catalog.UID,
				},
			},
		},
		// usefull ?
		Spec: imagev1.ImageStreamSpec{
			LookupPolicy: imagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}

	Log.Infof("Checking if Camel K builder container %s already exists...", imageName+":"+imageTag)
	// check if image is not snapshot and does exists
	if !imageSnapshot(imageName+":"+imageTag) && imageExistsS2I(ctx, c, is) {
		Log.Infof("S2I Camel K builder container %s already exists...", imageName+":"+imageTag)
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image exists on registry (later)",
		)
		target.Status.Image = imageName

		return target, nil
	} else {
		Log.Infof("S2I Camel K builder container %s does not exists or is snapshot...", imageName+":"+imageTag)
		/*target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			errors.New("S2I not complete, come back later"),
		)*/
	}

	// delete bc if exists ??
	Log.Infof("S2I Camel K builder container %s delete BuildConfig...", imageName+":"+imageTag)
	err := c.Delete(ctx, bc)
	if err != nil && !k8serrors.IsNotFound(err) {
		Log.Infof("S2I Camel K builder container %s cannot delete BuildConfig...", imageName+":"+imageTag)
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			fmt.Errorf("cannot delete BuildConfig: %w\n", err),
		)
		return target, fmt.Errorf("cannot delete BuildConfig: %w\n", err)
	}

	// create BuildConfig
	Log.Infof("S2I Camel K builder container %s create BuildConfig...", imageName+":"+imageTag)
	err = c.Create(ctx, bc)
	if err != nil {
		Log.Infof("S2I Camel K builder container %s cannot create BuildConfig...", imageName+":"+imageTag)
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			fmt.Errorf("cannot create BuildConfig: %w\n", err),
		)
		return target, fmt.Errorf("cannot create BuildConfig: %w\n", err)
	}

	// delete imagestream if exists ?
	Log.Infof("S2I Camel K builder container %s delete image stream...", imageName+":"+imageTag)
	err = c.Delete(ctx, is)
	if err != nil && !k8serrors.IsNotFound(err) {
		Log.Infof("S2I Camel K builder container %s cannot delete image stream...", imageName+":"+imageTag)
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			fmt.Errorf("cannot delete image stream: %w\n", err),
		)
		return target, fmt.Errorf("cannot delete image stream: %w\n", err)
	}

	//create image stream
	Log.Infof("S2I Camel K builder container %s create image stream...", imageName+":"+imageTag)
	err = c.Create(ctx, is)
	if err != nil {
		Log.Infof("S2I Camel K builder container %s cannot delete image stream...", imageName+":"+imageTag)
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			fmt.Errorf("cannot create image stream: %w\n", err),
		)
		return target, fmt.Errorf("cannot create image stream: %w\n", err)
	}

	Log.Infof("S2I Camel K builder container %s preparation done I GUESS...", imageName+":"+imageTag)

	// CREATE DA IMAGE
	err = util.WithTempDir(imageName+"-s2i-", func(tmpDir string) error {
		archive := filepath.Join(tmpDir, "archive.tar.gz")

		archiveFile, err := os.Create(archive)
		if err != nil {
			return fmt.Errorf("cannot create tar archive: %w", err)
		}

		err = tarFiles(archiveFile, "/usr/local/bin/kamel:/usr/local/bin/kamel",
			"/usr/share/maven/mvnw/:/usr/share/maven/mvnw/")
		if err != nil {
			return fmt.Errorf("cannot tar context directory: %w", err)
		}

		f, err := util.Open(archive)
		if err != nil {
			return err
		}

		restClient, err := apiutil.RESTClientForGVK(
			schema.GroupVersionKind{Group: "build.openshift.io", Version: "v1"}, false,
			c.GetConfig(), serializer.NewCodecFactory(c.GetScheme()))
		if err != nil {
			return err
		}

		r := restClient.Post().
			Namespace(bc.Namespace).
			Body(bufio.NewReader(f)).
			Resource("buildconfigs").
			Name(bc.Name).
			SubResource("instantiatebinary").
			Do(ctx)

		if r.Error() != nil {
			return fmt.Errorf("cannot instantiate binary: %w", err)
		}

		data, err := r.Raw()
		if err != nil {
			return fmt.Errorf("no raw data retrieved: %w", err)
		}

		s2iBuild := buildv1.Build{}
		err = json.Unmarshal(data, &s2iBuild)
		if err != nil {
			return fmt.Errorf("cannot unmarshal instantiated binary response: %w", err)
		}

		err = waitForS2iBuildCompletion(ctx, c, &s2iBuild)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				Log.Infof("Camel K builder container %s cancelled...", imageName+":"+imageTag)
				return fmt.Errorf("cannot create image stream: %w\n", err)
			}
		}
		if s2iBuild.Status.Output.To != nil {
			Log.Infof("S2I Camel K builder container %s builded with digest...", imageName+":"+imageTag+"@"+s2iBuild.Status.Output.To.ImageDigest)
		}

		err = c.Get(ctx, ctrl.ObjectKeyFromObject(is), is)
		if err != nil {
			return err
		}

		if is.Status.DockerImageRepository == "" {
			return errors.New("dockerImageRepository not available in ImageStream")
		}

		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image successfully built",
		)
		target.Status.Image = is.Status.DockerImageRepository + ":" + imageTag

		return f.Close()
	})

	if err != nil {
		Log.Infof("S2I Camel K builder container %s image creation error...", imageName+":"+imageTag)
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			fmt.Errorf("cannot create image: %w\n", err),
		)
		return target, fmt.Errorf("cannot create image: %w\n", err)
	}

	return target, nil
}

func initializeSpectrum(options spectrum.Options, ip *v1.IntegrationPlatform, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	target := catalog.DeepCopy()
	imageName := fmt.Sprintf(
		"%s/camel-k-runtime-%s-builder:%s",
		ip.Status.Build.Registry.Address,
		catalog.Spec.Runtime.Provider,
		strings.ToLower(catalog.Spec.Runtime.Version),
	)

	newStdR, newStdW, pipeErr := os.Pipe()
	defer util.CloseQuietly(newStdW)

	if pipeErr != nil {
		// In the unlikely case of an error, use stdout instead of aborting
		Log.Errorf(pipeErr, "Unable to remap I/O. Spectrum messages will be displayed on the stdout")
		newStdW = os.Stdout
	}
	go readSpectrumLogs(newStdR)

	// We use the future target image as a base just for the sake of pulling and verify it exists
	options.Base = imageName
	options.Stderr = newStdW
	options.Stdout = newStdW

	if !imageSnapshot(options.Base) && imageExistsSpectrum(options) {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image exists on registry",
		)
		target.Status.Image = imageName

		return target, nil
	}

	// Now we properly set the base and the target image
	options.Base = catalog.Spec.GetQuarkusToolingImage()
	options.Target = imageName

	err := buildRuntimeBuilderWithTimeout(options, ip.Status.Build.GetBuildCatalogToolTimeout().Duration)

	if err != nil {
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			err,
		)
	} else {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image successfully built",
		)
		target.Status.Image = imageName
	}

	return target, nil
}

func imageExistsSpectrum(options spectrum.Options) bool {
	Log.Infof("Checking if Camel K builder container %s already exists...", options.Base)
	ctrImg, err := spectrum.Pull(options)
	if ctrImg != nil && err == nil {
		var hash gcrv1.Hash
		if hash, err = ctrImg.Digest(); err != nil {
			Log.Errorf(err, "Cannot calculate digest")
			return false
		}
		Log.Infof("Found Camel K builder container with digest %s", hash.String())
		return true
	}

	Log.Infof("Couldn't pull image due to %s", err.Error())
	return false
}

func imageSnapshot(imageName string) bool {
	return strings.HasSuffix(imageName, "snapshot")
}

func buildRuntimeBuilderWithTimeout(options spectrum.Options, timeout time.Duration) error {
	// Backward compatibility with IP which had not a timeout field
	if timeout == 0 {
		return buildRuntimeBuilderImage(options)
	}
	result := make(chan error, 1)
	go func() {
		result <- buildRuntimeBuilderImage(options)
	}()
	select {
	case <-time.After(timeout):
		return fmt.Errorf("build timeout: %s", timeout.String())
	case result := <-result:
		return result
	}
}

// This func will take care to dynamically build an image that will contain the tools required
// by the catalog build plus kamel binary and a maven wrapper required for the build.
func buildRuntimeBuilderImage(options spectrum.Options) error {
	if options.Base == "" {
		return fmt.Errorf("missing base image, likely catalog is not compatible with this Camel K version")
	}
	Log.Infof("Making up Camel K builder container %s", options.Target)

	if jobs := runtime.GOMAXPROCS(0); jobs > 1 {
		options.Jobs = jobs
	}

	// TODO support also S2I
	_, err := spectrum.Build(options,
		"/usr/local/bin/kamel:/usr/local/bin/",
		"/usr/share/maven/mvnw/:/usr/share/maven/mvnw/")
	if err != nil {
		return err
	}

	return nil
}

func readSpectrumLogs(newStdOut io.Reader) {
	scanner := bufio.NewScanner(newStdOut)

	for scanner.Scan() {
		line := scanner.Text()
		Log.Infof(line)
	}
}

func makeSpectrumOptions(ctx context.Context, c client.Client, platformNamespace string, registry v1.RegistrySpec) (spectrum.Options, error) {
	options := spectrum.Options{}
	var err error
	registryConfigDir := ""
	if registry.Secret != "" {
		registryConfigDir, err = builder.MountSecret(ctx, c, platformNamespace, registry.Secret)
		if err != nil {
			return options, err
		}
	}
	options.PullInsecure = registry.Insecure
	options.PushInsecure = registry.Insecure
	options.PullConfigDir = registryConfigDir
	options.PushConfigDir = registryConfigDir
	options.Recursive = true

	return options, nil
}

func imageExistsS2I(ctx context.Context, c client.Client, is *imagev1.ImageStream) bool {

	key := k8sclient.ObjectKey{
		Namespace: is.Namespace,
		Name:      is.Name,
	}

	err := c.Get(ctx, key, is)

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			Log.Infof("Couldn't pull image due to %s", err.Error())
		}
		// DUMB LOG
		Log.Error(err, "error not found image")
		return false
	}
	return true
}

func tarFiles(writer io.Writer, files ...string) error {

	gzw := gzip.NewWriter(writer)
	defer util.CloseQuietly(gzw)

	tw := tar.NewWriter(gzw)
	defer util.CloseQuietly(tw)

	// Iterate over files and and add them to the tar archive
	for _, fileDetail := range files {
		log.Infof("file %s", fileDetail)
		fileSource := strings.Split(fileDetail, ":")[0]
		fileTarget := strings.Split(fileDetail, ":")[1]
		// ensure the src actually exists before trying to tar it
		if _, err := os.Stat(fileSource); err != nil {
			return fmt.Errorf("unable to tar files: %w", err)
		}

		if err := filepath.Walk(fileSource, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !fi.Mode().IsRegular() {
				return nil
			}

			header, err := tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}

			// update the name to correctly reflect the desired destination when un-taring
			header.Name = strings.TrimPrefix(strings.ReplaceAll(file, fileSource, fileTarget), string(filepath.Separator))

			log.Infof("file %s", file)
			log.Infof("header.Name %s", header.Name)

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			f, err := util.Open(file)
			if err != nil {
				return err
			}

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}

			return f.Close()
		}); err != nil {
			return fmt.Errorf("unable to tar: %w", err)
		}

	}
	return nil
}

func waitForS2iBuildCompletion(ctx context.Context, c client.Client, build *buildv1.Build) error {
	key := ctrl.ObjectKeyFromObject(build)
	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(1 * time.Second):
			err := c.Get(ctx, key, build)
			if err != nil {
				if k8serrors.IsNotFound(err) {
					continue
				}
				return err
			}

			if build.Status.Phase == buildv1.BuildPhaseComplete {
				return nil
			} else if build.Status.Phase == buildv1.BuildPhaseCancelled ||
				build.Status.Phase == buildv1.BuildPhaseFailed ||
				build.Status.Phase == buildv1.BuildPhaseError {
				return errors.New("build failed")
			}
		}
	}
}
