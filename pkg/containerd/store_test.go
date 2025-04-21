// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/azure/peerd/pkg/containerd/mocks"
	"github.com/containerd/containerd"
	eventtypes "github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/events"
	"github.com/containerd/containerd/images"
	"github.com/containerd/platforms"
	"github.com/containerd/typeurl/v2"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

func TestCreateFilter(t *testing.T) {
	tests := []struct {
		name                string
		hosts               []string
		expectedListFilter  string
		expectedEventFilter string
	}{
		{
			name:                "only registries",
			hosts:               []string{"https://docker.io", "https://gcr.io"},
			expectedListFilter:  `name~="docker.io|gcr.io"`,
			expectedEventFilter: `topic~="/images/create|/images/update",event.name~="docker.io|gcr.io"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedListFilter, getListFilter(tt.hosts))
			require.Equal(t, tt.expectedEventFilter, getEventFilter(tt.hosts))
		})
	}
}

func TestAllNoPlatform(t *testing.T) {
	cs := &mocks.MockContentStore{
		Data: map[string]string{
			// Index
			"sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a": `{ "mediaType": "application/vnd.oci.image.index.v1+json", "schemaVersion": 2, "manifests": [ { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "size": 2372, "platform": { "architecture": "amd64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "size": 2372, "platform": { "architecture": "arm", "os": "linux", "variant": "v7" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "size": 2372, "platform": { "architecture": "arm64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:73af5483f4d2d636275dcef14d5443ff96d7347a0720ca5a73a32c73855c4aac", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:36e11bf470af256febbdfad9d803e60b7290b0268218952991b392be9e8153bd", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:42d1c43f2285e8e3d39f80b8eed8e4c5c28b8011c942b5413ecc6a0050600609", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } } ] }`,
		},
	}

	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a")},
			},
		},
	}

	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs)))
	require.NoError(t, err)
	s := store{
		client:   client,
		platform: platforms.Only(platforms.MustParse("darwin/arm64")),
	}
	img, err := ParseReference("ghcr.io/distribution/distribution:v0.0.8", digest.Digest("sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a"))
	require.NoError(t, err)

	_, err = s.All(context.TODO(), img)
	require.EqualError(t, err, "failed to walk image manifests: could not find platform architecture in manifest: sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a")
}

func TestAll(t *testing.T) {
	tests := []struct {
		platformStr  string
		imageName    string
		imageDigest  string
		expectedKeys []string
	}{
		{
			platformStr: "linux/amd64",
			imageName:   "ghcr.io/distribution/distribution:v0.0.8-with-media-type",
			imageDigest: "sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
			expectedKeys: []string{
				"sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
				"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355",
				"sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e",
				"sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639",
				"sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58",
				"sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db",
				"sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265",
				"sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0",
				"sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c",
				"sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f",
				"sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c",
				"sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a",
				"sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef",
				"sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1",
			},
		},
		{
			platformStr: "linux/amd64",
			imageName:   "ghcr.io/distribution/distribution:v0.0.8-without-media-type",
			imageDigest: "sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
			expectedKeys: []string{
				"sha256:e2db0e6787216c5abfc42ea8ec82812e41782f3bc6e3b5221d5ef9c800e6c507",
				"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355",
				"sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e",
				"sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639",
				"sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58",
				"sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db",
				"sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265",
				"sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0",
				"sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c",
				"sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f",
				"sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c",
				"sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a",
				"sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef",
				"sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1",
			},
		},
		{
			platformStr: "linux/arm64",
			imageName:   "ghcr.io/distribution/distribution:v0.0.8-with-media-type",
			imageDigest: "sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
			expectedKeys: []string{
				"sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
				"sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53",
				"sha256:c73129c9fb699b620aac2df472196ed41797fd0f5a90e1942bfbf19849c4a1c9",
				"sha256:0b41f743fd4d78cb50ba86dd3b951b51458744109e1f5063a76bc5a792c3d8e7",
				"sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58",
				"sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db",
				"sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265",
				"sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0",
				"sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c",
				"sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f",
				"sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c",
				"sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a",
				"sha256:0dc769edeab7d9f622b9703579f6c89298a4cf45a84af1908e26fffca55341e1",
				"sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1",
			},
		},
		{
			platformStr: "linux/arm",
			imageName:   "ghcr.io/distribution/distribution:v0.0.8-with-media-type",
			imageDigest: "sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
			expectedKeys: []string{
				"sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
				"sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b",
				"sha256:1079836371d57a148a0afa5abfe00bd91825c869fcc6574a418f4371d53cab4c",
				"sha256:b437b30b8b4cc4e02865517b5ca9b66501752012a028e605da1c98beb0ed9f50",
				"sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58",
				"sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db",
				"sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265",
				"sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0",
				"sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c",
				"sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f",
				"sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c",
				"sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a",
				"sha256:01d28554416aa05390e2827a653a1289a2a549e46cc78d65915a75377c6008ba",
				"sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1",
			},
		},
	}

	cs := &mocks.MockContentStore{
		Data: map[string]string{
			// Index with media type
			"sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a": `{ "mediaType": "application/vnd.oci.image.index.v1+json", "schemaVersion": 2, "manifests": [ { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "size": 2372, "platform": { "architecture": "amd64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "size": 2372, "platform": { "architecture": "arm", "os": "linux", "variant": "v7" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "size": 2372, "platform": { "architecture": "arm64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:73af5483f4d2d636275dcef14d5443ff96d7347a0720ca5a73a32c73855c4aac", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:36e11bf470af256febbdfad9d803e60b7290b0268218952991b392be9e8153bd", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:42d1c43f2285e8e3d39f80b8eed8e4c5c28b8011c942b5413ecc6a0050600609", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } } ] }`,
			// Index without media type
			"sha256:e2db0e6787216c5abfc42ea8ec82812e41782f3bc6e3b5221d5ef9c800e6c507": `{ "schemaVersion": 2, "manifests": [ { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "size": 2372, "platform": { "architecture": "amd64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "size": 2372, "platform": { "architecture": "arm", "os": "linux", "variant": "v7" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "size": 2372, "platform": { "architecture": "arm64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:73af5483f4d2d636275dcef14d5443ff96d7347a0720ca5a73a32c73855c4aac", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:36e11bf470af256febbdfad9d803e60b7290b0268218952991b392be9e8153bd", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:42d1c43f2285e8e3d39f80b8eed8e4c5c28b8011c942b5413ecc6a0050600609", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } } ] }`,
			// AMD64
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef", "size": 36529876 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
			// ARM64
			"sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:c73129c9fb699b620aac2df472196ed41797fd0f5a90e1942bfbf19849c4a1c9", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:0b41f743fd4d78cb50ba86dd3b951b51458744109e1f5063a76bc5a792c3d8e7", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:0dc769edeab7d9f622b9703579f6c89298a4cf45a84af1908e26fffca55341e1", "size": 34168923 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
			// ARM
			"sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:1079836371d57a148a0afa5abfe00bd91825c869fcc6574a418f4371d53cab4c", "size": 2855 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b437b30b8b4cc4e02865517b5ca9b66501752012a028e605da1c98beb0ed9f50", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:01d28554416aa05390e2827a653a1289a2a549e46cc78d65915a75377c6008ba", "size": 34318536 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
		},
	}
	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8-with-media-type": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a")},
			},
			"ghcr.io/distribution/distribution:v0.0.8-without-media-type": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:e2db0e6787216c5abfc42ea8ec82812e41782f3bc6e3b5221d5ef9c800e6c507")},
			},
		},
	}
	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs)))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(strings.Join([]string{tt.platformStr, tt.imageName}, "-"), func(t *testing.T) {
			s := store{
				client:   client,
				platform: platforms.Only(platforms.MustParse(tt.platformStr)),
			}
			img, err := ParseReference(tt.imageName, digest.Digest(tt.imageDigest))
			require.NoError(t, err)

			keys, err := s.All(context.TODO(), img)
			require.NoError(t, err)
			require.Equal(t, tt.expectedKeys, keys)
		})
	}
}

func TestSize(t *testing.T) {
	cs := &mocks.MockContentStore{
		Data: map[string]string{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef", "size": 36529876 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
		},
	}
	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8-with-media-type": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355")},
			},
		},
	}

	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs)))
	require.NoError(t, err)

	for _, tt := range []struct {
		d           digest.Digest
		s           int64
		errExpected bool
	}{
		{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355",
			2062,
			false,
		},
		{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4353",
			0,
			true,
		},
	} {
		t.Run(fmt.Sprintf("%v-%v", tt.d.String(), tt.errExpected), func(t *testing.T) {
			s := store{
				client:   client,
				platform: platforms.Only(platforms.MustParse("linux/amd64")),
			}

			size, err := s.Size(context.Background(), tt.d)
			if tt.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.s, size)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	cs := &mocks.MockContentStore{
		Data: map[string]string{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef", "size": 36529876 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
		},
	}
	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355")},
			},
		},
	}

	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs)))
	require.NoError(t, err)

	for _, tt := range []struct {
		ref         string
		expected    digest.Digest
		errExpected bool
	}{
		{
			"ghcr.io/distribution/distribution:v0.0.8",
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355",
			false,
		},
		{
			"ghcr.io/distribution/distribution:latest",
			"",
			true,
		},
	} {
		t.Run(fmt.Sprintf("%v-%v", tt.ref, tt.errExpected), func(t *testing.T) {
			s := store{
				client:   client,
				platform: platforms.Only(platforms.MustParse("linux/amd64")),
			}

			got, err := s.Resolve(context.Background(), tt.ref)
			if tt.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestBytes(t *testing.T) {
	man := `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef", "size": 36529876 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`

	cs := &mocks.MockContentStore{
		Data: map[string]string{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355": man,
		},
	}
	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355")},
			},
		},
	}

	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs)))
	require.NoError(t, err)

	for _, tt := range []struct {
		d           digest.Digest
		expected    string
		mt          string
		errExpected bool
	}{
		{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355",
			man,
			"application/vnd.oci.image.manifest.v1+json",
			false,
		},
		{
			"ghcr.io/distribution/distribution:latest",
			"",
			"",
			true,
		},
	} {
		t.Run(fmt.Sprintf("%v-%v", tt.d, tt.errExpected), func(t *testing.T) {
			s := store{
				client:   client,
				platform: platforms.Only(platforms.MustParse("linux/amd64")),
			}

			got, mt, err := s.Bytes(context.Background(), tt.d)
			if tt.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, string(got))
				require.Equal(t, tt.mt, mt)
			}
		})
	}
}

func TestGetHostNames(t *testing.T) {
	for _, tt := range []struct {
		hosts    []string
		expected []string
	}{
		{
			hosts:    []string{"ghcr.io"},
			expected: []string{"ghcr.io"},
		},
		{
			hosts:    []string{"ghcr.io", "docker.io", "mcr.microsoft.com", "localhost:5000"},
			expected: []string{"ghcr.io", "docker.io", "mcr.microsoft.com", "localhost:5000"},
		},
		{
			hosts:    []string{"https://k8s.io", "https://registry-1.docker.io"},
			expected: []string{"k8s.io", "registry-1.docker.io"},
		},
	} {
		t.Run(fmt.Sprintf("%v", tt.hosts), func(t *testing.T) {
			got := getHostNames(tt.hosts)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestList(t *testing.T) {
	cs := &mocks.MockContentStore{
		Data: map[string]string{
			// Index with media type
			"sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a": `{ "mediaType": "application/vnd.oci.image.index.v1+json", "schemaVersion": 2, "manifests": [ { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "size": 2372, "platform": { "architecture": "amd64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "size": 2372, "platform": { "architecture": "arm", "os": "linux", "variant": "v7" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "size": 2372, "platform": { "architecture": "arm64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:73af5483f4d2d636275dcef14d5443ff96d7347a0720ca5a73a32c73855c4aac", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:36e11bf470af256febbdfad9d803e60b7290b0268218952991b392be9e8153bd", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:42d1c43f2285e8e3d39f80b8eed8e4c5c28b8011c942b5413ecc6a0050600609", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } } ] }`,
			// Index without media type
			"sha256:e2db0e6787216c5abfc42ea8ec82812e41782f3bc6e3b5221d5ef9c800e6c507": `{ "schemaVersion": 2, "manifests": [ { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "size": 2372, "platform": { "architecture": "amd64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "size": 2372, "platform": { "architecture": "arm", "os": "linux", "variant": "v7" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "size": 2372, "platform": { "architecture": "arm64", "os": "linux" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:73af5483f4d2d636275dcef14d5443ff96d7347a0720ca5a73a32c73855c4aac", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:36e11bf470af256febbdfad9d803e60b7290b0268218952991b392be9e8153bd", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } }, { "mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": "sha256:42d1c43f2285e8e3d39f80b8eed8e4c5c28b8011c942b5413ecc6a0050600609", "size": 566, "annotations": { "vnd.docker.reference.digest": "sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53", "vnd.docker.reference.type": "attestation-manifest" }, "platform": { "architecture": "unknown", "os": "unknown" } } ] }`,
			// AMD64
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef", "size": 36529876 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
			// ARM64
			"sha256:dce623533c59af554b85f859e91fc1cbb7f574e873c82f36b9ea05a09feb0b53": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:c73129c9fb699b620aac2df472196ed41797fd0f5a90e1942bfbf19849c4a1c9", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:0b41f743fd4d78cb50ba86dd3b951b51458744109e1f5063a76bc5a792c3d8e7", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:0dc769edeab7d9f622b9703579f6c89298a4cf45a84af1908e26fffca55341e1", "size": 34168923 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
			// ARM
			"sha256:0ad7c556c55464fa44d4c41e5236715e015b0266daced62140fb5c6b983c946b": `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:1079836371d57a148a0afa5abfe00bd91825c869fcc6574a418f4371d53cab4c", "size": 2855 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b437b30b8b4cc4e02865517b5ca9b66501752012a028e605da1c98beb0ed9f50", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:01d28554416aa05390e2827a653a1289a2a549e46cc78d65915a75377c6008ba", "size": 34318536 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`,
		},
	}
	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8-with-media-type": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a")},
				Name:   "ghcr.io/distribution/distribution:v0.0.8-with-media-type",
			},
			"mcr.microsoft.com/distribution/distribution:v0.0.8-without-media-type": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:e2db0e6787216c5abfc42ea8ec82812e41782f3bc6e3b5221d5ef9c800e6c507")},
				Name:   "mcr.microsoft.com/distribution/distribution:v0.0.8-without-media-type",
			},
		},
	}

	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs)))
	require.NoError(t, err)

	for _, tt := range []struct {
		hosts       []string
		expected    []Reference
		errExpected bool
	}{
		{
			[]string{"ghcr.io"},
			[]Reference{
				&reference{
					name: "ghcr.io/distribution/distribution:v0.0.8-with-media-type",
					host: "ghcr.io",
					tag:  "v0.0.8-with-media-type",
					dgst: "sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
					repo: "distribution/distribution",
				},
			},
			false,
		},
		{
			[]string{"mcr.microsoft.com"},
			[]Reference{
				&reference{
					name: "mcr.microsoft.com/distribution/distribution:v0.0.8-without-media-type",
					host: "mcr.microsoft.com",
					tag:  "v0.0.8-without-media-type",
					dgst: "sha256:e2db0e6787216c5abfc42ea8ec82812e41782f3bc6e3b5221d5ef9c800e6c507",
					repo: "distribution/distribution",
				},
			},
			false,
		},
		{
			[]string{"ghcr.io", "mcr.microsoft.com"},
			[]Reference{
				&reference{
					name: "ghcr.io/distribution/distribution:v0.0.8-with-media-type",
					host: "ghcr.io",
					tag:  "v0.0.8-with-media-type",
					dgst: "sha256:e80e36564e9617f684eb5972bf86dc9e9e761216e0d40ff78ca07741ec70725a",
					repo: "distribution/distribution",
				},
				&reference{
					name: "mcr.microsoft.com/distribution/distribution:v0.0.8-without-media-type",
					host: "mcr.microsoft.com",
					tag:  "v0.0.8-without-media-type",
					dgst: "sha256:e2db0e6787216c5abfc42ea8ec82812e41782f3bc6e3b5221d5ef9c800e6c507",
					repo: "distribution/distribution",
				},
			},
			false,
		},
	} {
		t.Run(fmt.Sprintf("%v-%v", tt.hosts, tt.errExpected), func(t *testing.T) {
			s, err := newStore(tt.hosts, client)
			require.NoError(t, err)

			got, err := s.List(context.Background())
			if tt.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	man := `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef", "size": 36529876 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`

	cs := &mocks.MockContentStore{
		Data: map[string]string{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355": man,
		},
	}
	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355")},
			},
		},
	}

	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs)))
	require.NoError(t, err)

	for _, tt := range []struct {
		d           digest.Digest
		expected    string
		errExpected bool
	}{
		{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355",
			man,
			false,
		},
		{
			"ghcr.io/distribution/distribution:latest",
			"",
			true,
		},
	} {
		t.Run(fmt.Sprintf("%v-%v", tt.d, tt.errExpected), func(t *testing.T) {
			s := store{
				client:   client,
				platform: platforms.Only(platforms.MustParse("linux/amd64")),
			}

			var buf bytes.Buffer

			err := s.Write(context.Background(), &buf, tt.d)
			if tt.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, buf.String())
			}
		})
	}
}

func TestSubscribe(t *testing.T) {
	man := `{ "mediaType": "application/vnd.oci.image.manifest.v1+json", "schemaVersion": 2, "config": { "mediaType": "application/vnd.oci.image.config.v1+json", "digest": "sha256:d715ba0d85ee7d37da627d0679652680ed2cb23dde6120f25143a0b8079ee47e", "size": 2842 }, "layers": [ { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:a7ca0d9ba68fdce7e15bc0952d3e898e970548ca24d57698725836c039086639", "size": 103732 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58", "size": 21202 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:b02a7525f878e61fc1ef8a7405a2cc17f866e8de222c1c98fd6681aff6e509db", "size": 716491 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265", "size": 317 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0", "size": 198 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c", "size": 113 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f", "size": 385 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c", "size": 355 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a", "size": 130562 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:76f3a495ffdc00c612747ba0c59fc56d0a2610d2785e80e9edddbf214c2709ef", "size": 36529876 }, { "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1", "size": 32 } ] }`

	cs := &mocks.MockContentStore{
		Data: map[string]string{
			"sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355": man,
		},
	}
	is := &mocks.MockImageStore{
		Data: map[string]images.Image{
			"ghcr.io/distribution/distribution:v0.0.8": {
				Target: ocispec.Descriptor{MediaType: "application/vnd.oci.image.index.v1+json", Digest: digest.Digest("sha256:44cb2cf712c060f69df7310e99339c1eb51a085446f1bb6d44469acff35b4355")},
				Name:   "ghcr.io/distribution/distribution:v0.0.8",
			},
		},
	}

	es := &mocks.MockEventService{
		EnvelopeChan: make(chan *events.Envelope),
		ErrorsChan:   make(chan error),
	}

	client, err := containerd.New("", containerd.WithServices(containerd.WithImageStore(is), containerd.WithContentStore(cs), containerd.WithEventService(es)))
	require.NoError(t, err)

	s, err := newStore([]string{"ghcr.io"}, client)
	require.NoError(t, err)

	gotEnvCh, gotErrCh := s.Subscribe(context.Background())
	require.NotNil(t, gotEnvCh)
	require.NotNil(t, gotErrCh)

	testDone := make(chan struct{})
	defer func() {
		testDone <- struct{}{}
		close(testDone)
	}()
	errorCount := 0
	eventsCount := 0
	totalCount := 0

	go func() {
		for {
			select {
			case <-gotEnvCh:
				totalCount++
				eventsCount++

				if eventsCount > 1 {
					t.Errorf("got %d events, want 1", eventsCount)
				}

			case e := <-gotErrCh:
				totalCount++
				errorCount++

				// Expect one error for the unexpected event.
				if errorCount > 1 {
					t.Errorf("got %d errors, want 1: %v", errorCount, e)
				}

			case <-testDone:
				return
			}
		}
	}()

	// Send an unexpected event.
	delEvent := eventtypes.ImageDelete{Name: "ghcr.io/distribution/distribution:v0.0.8"}
	delAny, err := typeurl.MarshalAny(&delEvent)
	require.NoError(t, err)
	go func() {
		es.EnvelopeChan <- &events.Envelope{
			Timestamp: time.Time{},
			Namespace: DefaultNamespace,
			Topic:     "unexpected",
			Event:     delAny,
		}
	}()

	for totalCount < 1 {
		time.Sleep(10 * time.Millisecond)
	}

	require.Equal(t, 1, errorCount)
	require.Equal(t, 0, eventsCount)
	require.Equal(t, 1, totalCount)

	// Send an image create event.
	createEvent := eventtypes.ImageCreate{Name: "ghcr.io/distribution/distribution:v0.0.8"}
	createAny, err := typeurl.MarshalAny(&createEvent)
	require.NoError(t, err)
	go func() {
		es.EnvelopeChan <- &events.Envelope{
			Timestamp: time.Time{},
			Namespace: DefaultNamespace,
			Topic:     "image-create",
			Event:     createAny,
		}
	}()

	for totalCount < 2 {
		time.Sleep(10 * time.Millisecond)
	}

	require.Equal(t, 1, errorCount) // no new error
	require.Equal(t, 1, eventsCount)
	require.Equal(t, 2, totalCount)
}
