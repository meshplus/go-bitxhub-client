package rpcx

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/meshplus/bitxhub-model/pb"
)

// IPFSPutFromLocal puts local file to ipfs
// args@localPath e.g. /tmp/eg.json
// returns cid of file stored on ipfs
func (cli *ChainClient) IPFSPutFromLocal(localfPath string) (*pb.Response, error) {
	res, err := cli.ipfsClient.PutFromLocal(localfPath)
	if err != nil {
		return nil, err
	}
	return &pb.Response{Data: res}, nil
}

// IPFSGet gets from ipfs
// args@path e.g. /ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme
func (cli *ChainClient) IPFSGet(path string) (*pb.Response, error) {
	res, err := cli.ipfsClient.Get(path)
	if err != nil {
		return nil, err
	}
	return &pb.Response{Data: res}, nil
}

// IPFSGetToLocal gets from ipfs and saves to local file path
// args@path e.g. /ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme
// args@localPath e.g. /tmp/readme
func (cli *ChainClient) IPFSGetToLocal(path string, localfPath string) (*pb.Response, error) {
	err := cli.ipfsClient.GetToLocal(path, localfPath)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// IPFSClient .
type IPFSClient struct {
	// apiAddrs  []string
	apiShells map[string]*shell.Shell
}

// NewIPFSClient .
func NewIPFSClient(options ...func(*IPFSClient)) (*IPFSClient, error) {
	m := make(map[string]*shell.Shell)
	c := &IPFSClient{
		apiShells: m,
	}
	for _, option := range options {
		option(c)
	}
	return c, nil
}

// WithAPIAddrs .
// e.g []string{"http://localhost:5001"}
func WithAPIAddrs(addrs []string) func(*IPFSClient) {
	return func(i *IPFSClient) {
		for _, addr := range addrs {
			i.AddAPIShell(addr)
		}
	}
}

// AddAPIShell add ipfs api address
func (ipfsClient *IPFSClient) AddAPIShell(addr string) {
	ipfsClient.apiShells[addr] = shell.NewShell(addr)
}

// RmAPIAddr rm ipfs api address
func (ipfsClient *IPFSClient) RmAPIAddr(addr string) {
	delete(ipfsClient.apiShells, addr)
}

// IPFSResponse describes ipfs add response
type IPFSResponse struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"`
	Size string `json:"Size"`
}

// PutFromLocal puts local file to ipfs
// args@localPath e.g. /tmp/eg.json
// returns cid of file stored on ipfs
func (ipfsClient *IPFSClient) PutFromLocal(localfPath string) ([]byte, error) {
	if len(ipfsClient.apiShells) <= 0 {
		return nil, fmt.Errorf("api shells nil")
	}

	var shells []*shell.Shell
	for _, shell := range ipfsClient.apiShells {
		shells = append(shells, shell)
	}
	limit := uint(len(shells) - 1)

	var response string
	err := retry.Retry(func(attempt uint) error {
		localFile, err := os.Open(localfPath)
		if err != nil {
			return err
		}
		response, err = shells[attempt].Add(localFile)
		if err != nil {
			return err
		}
		localFile.Close()
		return nil
	},
		strategy.Limit(limit),
	)

	if err != nil {
		return nil, err
	}

	return []byte(response), nil
}

// Get gets from ipfs
// args@path e.g. /ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme
// returns content of file
func (ipfsClient *IPFSClient) Get(path string) ([]byte, error) {
	if len(ipfsClient.apiShells) <= 0 {
		return nil, fmt.Errorf("api address nil")
	}

	var shells []*shell.Shell
	for _, shell := range ipfsClient.apiShells {
		shells = append(shells, shell)
	}
	limit := uint(len(shells) - 1)

	var response []byte
	err := retry.Retry(func(attempt uint) error {
		res, err := shells[attempt].Cat(path)
		if err != nil {
			return err
		}
		defer res.Close()
		response, err = ioutil.ReadAll(res)
		if err != nil {
			return err
		}
		return nil
	},
		strategy.Limit(limit),
	)

	if err != nil {
		return nil, err
	}
	return response, err
}

// GetToLocal gets from ipfs and saves to local file path
// args@path e.g. /ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme
// args@localPath e.g. /tmp/readme
func (ipfsClient *IPFSClient) GetToLocal(path string, localfPath string) error {
	content, err := ipfsClient.Get(path)
	if err != nil {
		return err
	}

	f, err := os.Create(localfPath)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	_, err = w.WriteString(string(content))
	if err != nil {
		return err
	}
	w.Flush()
	f.Close()
	return nil
}
