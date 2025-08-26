package forge

import (
	"io"
	"os"

	"code.gitea.io/sdk/gitea"
	eve "eve.evalgo.org/common"
)

func GiteaGetRepo(url, token, owner, repo, branch string) {
	client, err := gitea.NewClient(url, gitea.SetToken(token))
	if err != nil {
		eve.Logger.Fatal("Gitea error: ", err)
	}
	reader, resp, err := client.GetArchiveReader(owner, repo, branch, gitea.TarGZArchive)
	if err != nil {
		eve.Logger.Fatal("Gitea error: ", err)
	}
	defer resp.Body.Close()
	out, err := os.Create(repo + "-" + branch + ".tar.gz")
	if err != nil {
		eve.Logger.Fatal("Gitea error: ", err)
	}
	defer out.Close()
	if _, err = io.Copy(out, reader); err != nil {
		eve.Logger.Fatal("Gitea error: ", err)
	}
}
