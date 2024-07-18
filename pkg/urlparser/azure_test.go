// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package urlparser

import (
	"testing"

	"github.com/opencontainers/go-digest"
)

var (
	azureTestCases = []struct {
		url    string
		digest string
		valid  bool
	}{
		{
			"",
			"",
			false,
		},
		{
			"https://aviral26.eastus.data.azurecr.io?t=A2DC7F4DA8D829EB400019A84F4EF18C88652134&h=aviral26.azurecr.io&c=3d109b5a-6a3e-48fe-aa7f-0c02abc773de&r=spegel&d=sha256:dd5ad9c9c29f04b41a0155c720cf5ccab28ef6d353f1fe17a06c579c70054f0a&p=myOe89N4rv2atPh7zOrPqZAm-hE-ySHDGacBDMzEMkYZP2IAPWnjkZdWyXOi6bpCmplEOw7UwesbL46W7u1UXujVN8rmsw-qp3N0l0U79_Uy4T2ajdvZCE2tvP00zXDgBlEW3J9A-78-P2wOqfwaocBBJJFYjTIgtN82pOBX-mqdqeNqv4a9cykSMLGdicfHINC6H9lXCbowKJNB4sv-PXnl0aL02OhgrB4Ki2F7szOMGYaOG1DEXwWYpn1mYWhytMU8kBPnqvVS39Yo-umiq6A7zJnhkYlVGIzDeOWd-OCV-qfrD4AWjJK8WFv986KlWam5kjs9n-dQetKN9eclNNNEbwvqEV_7pRTvTXUsMNR-BqqTTumUAjB8nII8h18gabzAuN80s1oG4ZF9VeZuFKIeCGlhZj1LwvMays7TV_ILCAcNyexshWI3tWSfrdotK8-LYINqW_pD63iMJBShb1-EZWzSd_mOcYrBHViQaFf_-3qI14aqNrL7ASGb3rzmizH1dFqphsYm7ltQ60CY18zbugsFCob-6yWFggpv6NlJ7ko5B-sT8VY0ljH1zHEFtOvf32pDVKR2hsOJwilpF0yzFSy0_di1OkjmIYnChxovvaCSpiMACRh_N5OPT29D&s=dyqCD1Q1d278Z_4nuZn7k7WcBnbq-D5p0kllhH7LrB8uvcxCrEBydZFZ5fe6UOp30kjmKodMvW88eVoWmNNrvKnRwkuKL9ZkHULPzUHqrVnH8rPZb2GrsUpFVDszPXTt8Z9eptNOCWj9jq15fMW6aWIlYEHC81fHrx_XBU1y6Sg0Bl2scQp0TFxvkl_SR64yzcRrUPMOAfgTLe9ILXZIaagkoEzpgyWk-AwIedBjP9X3Y_yZmMvDb6IPL6trC3rh8qyfG09VmSkWLyAx3OwFtKyk4IK4BNgAB-kg2SHaoQ67jJLf-lR3CDRu6HvJNIa3_7gPBUZMQmpsE7rYvMCNZQ&v=1",
			"sha256:dd5ad9c9c29f04b41a0155c720cf5ccab28ef6d353f1fe17a06c579c70054f0a",
			true,
		},
		{
			"https://westus2.data.mcr.microsoft.com/01031d61e1024861afee5d512651eb9f-h36fskt2ei//docker/registry/v2/blobs/sha256/1b/1b930d010525941c1d56ec53b97bd057a67ae1865eebf042686d2a2d18271ced/data?se=2023-09-20T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f",
			"sha256:1b930d010525941c1d56ec53b97bd057a67ae1865eebf042686d2a2d18271ced",
			true,
		},
		{
			"https://eusreplstore28.blob.core.windows.net/dd5ad9c9c29f04b4-46d325e77acf422cbc239cd963f8d78d-4643a09878//docker/registry/v2/blobs/sha256/dd/dd5ad9c9c29f04b41a0155c720cf5ccab28ef6d353f1fe17a06c579c70054f0a/data?se=2023-09-20T01%3A15%3A41Z&sig=6V%2FV9T7i373TPyxD4dzXlN16KzEW3GchbULPHg2EKjE%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=46d325e77acf422cbc239cd963f8d78d",
			"sha256:dd5ad9c9c29f04b41a0155c720cf5ccab28ef6d353f1fe17a06c579c70054f0a",
			true,
		},
		{
			"https://aviral26.eastus.data.azurecr.io?t=A2DC7F4DA8D829EB400019A84F4EF18C88652134&h=aviral26.azurecr.io&c=3d109b5a-6a3e-48fe-aa7f-0c02abc773de&r=spegel&d=sha512:dd5ad9c9c29f04b41a0155c720cf5ccab28ef6d353f1fe17a06c579c70054f0a&p=myOe89N4rv2atPh7zOrPqZAm-hE-ySHDGacBDMzEMkYZP2IAPWnjkZdWyXOi6bpCmplEOw7UwesbL46W7u1UXujVN8rmsw-qp3N0l0U79_Uy4T2ajdvZCE2tvP00zXDgBlEW3J9A-78-P2wOqfwaocBBJJFYjTIgtN82pOBX-mqdqeNqv4a9cykSMLGdicfHINC6H9lXCbowKJNB4sv-PXnl0aL02OhgrB4Ki2F7szOMGYaOG1DEXwWYpn1mYWhytMU8kBPnqvVS39Yo-umiq6A7zJnhkYlVGIzDeOWd-OCV-qfrD4AWjJK8WFv986KlWam5kjs9n-dQetKN9eclNNNEbwvqEV_7pRTvTXUsMNR-BqqTTumUAjB8nII8h18gabzAuN80s1oG4ZF9VeZuFKIeCGlhZj1LwvMays7TV_ILCAcNyexshWI3tWSfrdotK8-LYINqW_pD63iMJBShb1-EZWzSd_mOcYrBHViQaFf_-3qI14aqNrL7ASGb3rzmizH1dFqphsYm7ltQ60CY18zbugsFCob-6yWFggpv6NlJ7ko5B-sT8VY0ljH1zHEFtOvf32pDVKR2hsOJwilpF0yzFSy0_di1OkjmIYnChxovvaCSpiMACRh_N5OPT29D&s=dyqCD1Q1d278Z_4nuZn7k7WcBnbq-D5p0kllhH7LrB8uvcxCrEBydZFZ5fe6UOp30kjmKodMvW88eVoWmNNrvKnRwkuKL9ZkHULPzUHqrVnH8rPZb2GrsUpFVDszPXTt8Z9eptNOCWj9jq15fMW6aWIlYEHC81fHrx_XBU1y6Sg0Bl2scQp0TFxvkl_SR64yzcRrUPMOAfgTLe9ILXZIaagkoEzpgyWk-AwIedBjP9X3Y_yZmMvDb6IPL6trC3rh8qyfG09VmSkWLyAx3OwFtKyk4IK4BNgAB-kg2SHaoQ67jJLf-lR3CDRu6HvJNIa3_7gPBUZMQmpsE7rYvMCNZQ&v=1",
			"",
			false,
		},
		{
			"https://westus2.data.mcr.microsoft.com/01031d61e1024861afee5d512651eb9f-h36fskt2ei//docker//v2/blobs/sha256/1b/1b930d010525941c1d56ec53b97bd057a67ae1865eebf042686d2a2d18271ced/data?se=2023-09-20T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f",
			"",
			false,
		},
		{
			"https://eusreplstore28.blob.core.windows.net/dd5ad9c9c29f04b4-46d325e77acf422cbc239cd963f8d78d-4643a09878//docker/registry/v2/blobs/sha256/dd/data?se=2023-09-20T01%3A15%3A41Z&sig=6V%2FV9T7i373TPyxD4dzXlN16KzEW3GchbULPHg2EKjE%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=46d325e77acf422cbc239cd963f8d78d",
			"",
			false,
		},
		{
			"https://aviral26.eastus.data.azurecr.io/?t=63F0223FE7717ED2EF4FD672251A1E3A62E8CD88&h=aviral26.azurecr.io&c=114c2ff3-318c-4f00-86b0-91ea33a781a4&r=scanner&d=sha256:526e51f1c889e6e9ef66e398aac57c388d1305c2490f0d235ff9c8346ab42ec1&p=-qVpMgk5_1AOKyTBbUmjdL4jTWrEGDMjmIJC60c8PuIDY7rIDOa9mkmP-b5OrHfnkNf9E6dDMycKey89jmOA5n3og-O6sd0fckVKJHLm6EoeAHi8qWNKA6AjshSRFN3CvcM9dXdmVpCmR1cR0Q-Cn9tPwyYpRNwQYdkbC5K17RdFUNVTfSETnrKGp5q0fM5WJRqJjB1D2XCSABvyw4Xp6Urp9fj-CcL1qI7DOef_hyHoktZyz47FQyRCQNaNpctoZJcQHjauC2i3zb_V5UIV4Bvy8ag6dfltxT7Z_tM3jpfuyv7dd-l--j3vMsvSp2KB_80WNdJPOpy_wMBSkOns3K_6DpmHdl7bO7keY6-UUYnx6c9nhYRDw7YZ6TsU5kFXndIJAzqfb87DGAmVEPPDJqvWYViL2NiPn_hmeBtre8ti26gVjITkdrrrkAt0TlIzG1fQeYsKKml49Dn-ftpkqzGWTT3soTO8TpGesx6sooCnpAkOsFWLd_pKBIzq-UvQwvJzLVVtf_tiB74ZmLrTCgXgm61q0Lqtq2lgc8zMY6kCQJl-BeycSWlxoEaNMgiLD4UaOOkoeuyOsJGQ-LxPJtVjxu0cuE41lppK_sijGpNylKnECs_cKKhIARlVrrY6kn63qEjQaW6CVMAbylVI8w&s=pBKH0rivCRPRzKcJhLm_D1xApjOwME1BsWEgATUSlL67R_rSbR218vQtoeca4nvuVMNlVGTxhElYhKuWYeVvx57wwHQ-3byGRhhM5FAKozixJpTli2O7oCSou8DRK1NoDATKLNGQ_RPgpZsXlrqQEIZNTn2U1rh5m6jwiKcyGBUX-R3BnymIt-kESrY48Z5u2Ey8orP6zXPupfb8S_pW3p5rfHk0fK9d6VzfZrVXTZbbQyIKXCRupZgVm7Uqvz3j0UTagYR-sy0-aec5ex639-GoHjTmzgwDVXVJkFV11fTw4CBHZtkjeFvXPiKV8ZlxlA0E-lBbaSRgU15ftSf04Q&v=1&l=false",
			"sha256:526e51f1c889e6e9ef66e398aac57c388d1305c2490f0d235ff9c8346ab42ec1",
			true,
		},
	}
)

func TestUrls(t *testing.T) {
	for _, test := range azureTestCases {
		got, err := parseDigestFromAzureUrl(test.url)
		if test.valid {
			if err != nil {
				t.Errorf("expected no error parsing digest from url %s", test.url)
			} else if got != digest.Digest(test.digest) {
				t.Errorf("expected digest %s, got %s", test.digest, got)
			}
		} else {
			if err == nil {
				t.Errorf("expected error parsing digest from url %s", test.url)
			}
		}
	}
}
