package eks

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

// New client to generate
func New(cluster *types.Cluster, tok token.Token) (*kubernetes.Clientset, error) {
	ca, err := base64.StdEncoding.DecodeString(aws.ToString(cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(
		&rest.Config{
			Host:        aws.ToString(cluster.Endpoint),
			BearerToken: tok.Token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: ca,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// GeneratorType is used
type GeneratorType struct {
	Context context.Context
}

// Generator interface which describes the k8s master API server we connect to
type Generator interface {
	DescribeCluster(*eks.Client, string) (*eks.DescribeClusterOutput, error)
	GenToken(*string) (token.Token, error)
	TestCluster(*types.Cluster, token.Token) error
	WriteCA(AwsConfig, *eks.DescribeClusterOutput) string
	// NewGenerator(bool, bool) (token.Generator, error)
	// GetWithOptions(*token.GetTokenOptions) (token.Token, error)
}

// DescribeCluster calls the AWS SDK EKS API to get details on the cluster
func (g *GeneratorType) DescribeCluster(e *eks.Client, cluster string) (*eks.DescribeClusterOutput, error) {
	return e.DescribeCluster(g.Context, &eks.DescribeClusterInput{Name: &cluster})
}

// TestCluster is a tough function to test as it makes a call to Nodes().
func (g *GeneratorType) TestCluster(r *types.Cluster, tok token.Token) error {
	clientset, err := New(r, tok)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}
	nodes, err := clientset.CoreV1().Nodes().List(g.Context, metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting EKS nodes: %v", err)
		return err
	}
	log.Debugf("There are %d nodes associated with cluster %s", len(nodes.Items), *r.Name)
	return nil
}

// GenToken will get the EKS cluster oauth2 token.
// Consider refresh flow instead and make token private
// and accessible via function call.
func (g *GeneratorType) GenToken(cluster *string) (token.Token, error) {
	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return token.Token{}, err
	}
	opts := &token.GetTokenOptions{
		// AssumeRoleARN: "arn:aws:iam::<account_id>:role/<role-name>", // Consider supporting this via config...
		ClusterID:     aws.ToString(cluster),
		Region:        "ap-southeast-2",
		AssumeRoleARN: os.Getenv("ROLE_ARN"),
	}
	return gen.GetWithOptions(opts)
}

// WriteCA - need to pass in another interface if i wanna mock this.
func (g *GeneratorType) WriteCA(a AwsConfig, e *eks.DescribeClusterOutput) string {
	file, err := ioutil.TempFile(a.J.TmpFileLocation, a.J.TmpFilePrefix)
	if err != nil {
		log.Fatal(err)
	}
	decoded, _ := base64.StdEncoding.DecodeString(*e.Cluster.CertificateAuthority.Data)
	file.Write([]byte(decoded))
	if err = file.Close(); err != nil {
		log.Fatal(err)
	}
	return file.Name()
}
