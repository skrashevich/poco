// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/mudler/luet/pkg/api/core/image"
	"github.com/otiai10/copy"
	"github.com/pterm/pterm"
	"github.com/skrashevich/poco/internal"
	"github.com/skrashevich/poco/pkg/bundler"
	"github.com/skrashevich/poco/pkg/extractor"
	"github.com/urfave/cli"
)

func common() []cli.Flag {

	return []cli.Flag{
		&cli.StringFlag{
			Name:   "entrypoint",
			EnvVar: "ENTRYPOINT",
			Usage:  "Default binary entrypoint. This is the first binary from the container image which will be executed.",
			Value:  "/bin/sh",
		},
		&cli.StringFlag{
			Name:   "output",
			Usage:  "Default binary output location",
			EnvVar: "OUTPUT",
			Value:  "sample",
		},
		&cli.StringFlag{
			Name:   "directory",
			Usage:  "Directory to pack",
			EnvVar: "DIRECTORY",
			Value:  "",
		},
		&cli.StringFlag{
			Name:   "app-description",
			Usage:  "App description",
			EnvVar: "DESCRIPTION",
			Value:  "sample",
		},
		&cli.StringFlag{
			Name:   "app-copyright",
			Usage:  "App copyright",
			EnvVar: "COPYRIGHT",
			Value:  "sample",
		},
		&cli.StringFlag{
			Name:   "app-author",
			Usage:  "App author",
			EnvVar: "AUTHOR",
			Value:  "sample",
		},
		&cli.StringFlag{
			Name:   "app-name",
			Usage:  "Application name",
			EnvVar: "NAME",
			Value:  "sample",
		},
		&cli.StringFlag{
			Name:   "app-version",
			Usage:  "Application version. This is used during bundle upgrades.",
			EnvVar: "VERSION",
			Value:  "0.1",
		},
		&cli.BoolFlag{
			Name:   "local",
			EnvVar: "LOCAL",
			Usage:  "Use local docker daemon to retrieve the image",
		},
		&cli.StringSliceFlag{
			Name:   "app-mounts",
			Usage:  "Define a list of default application mount bindings. For example: /tmp, /dev:/foo/dev",
			EnvVar: "MOUNTS",
		},
		&cli.StringSliceFlag{
			Name:   "app-attrs",
			Usage:  "Define a list of default application attrs. Example: ipc, net, user",
			EnvVar: "ATTRS",
			Value:  &cli.StringSlice{"ipc", "uts", "user", "ns", "pid"},
		},
		&cli.StringSliceFlag{
			Name:   "app-store",
			Usage:  "Define a default application store where the bundle content will be uncompressed. It defaults to a temporary directory otherwise. (e.g. $HOME/.app/foo)",
			EnvVar: "STORE",
		},
		&cli.StringFlag{
			Usage:  "Image to be used as bundle content",
			Name:   "image",
			EnvVar: "IMAGE",
			Value:  "alpine",
		},
		&cli.StringFlag{
			Name:   "command-prefix",
			EnvVar: "COMMAND_PREFIX",
			Value:  "sudo",
			Usage:  "Prefix go generate commands with sudo. This is required if not running bundler as root and want to preserve container permissions",
		},
	}
}

func pocoVersion() string {
	return fmt.Sprintf("%s-g%s", internal.Version, internal.Commit)
}

func cliParse(c *cli.Context) *bundler.Bundler {
	opts := []bundler.Option{
		bundler.WithRenderData(
			c.String("image"),
			c.String("command-prefix"),
			c.Bool("local"),
			c.String("directory") == "",
			bundler.App{
				Name:        c.String("app-name"),
				Author:      c.String("app-author"),
				Version:     c.String("app-version"),
				Entrypoint:  c.String("entrypoint"),
				Mounts:      c.StringSlice("app-mounts"),
				Attrs:       c.StringSlice("app-attrs"),
				Copyright:   c.String("app-copyright"),
				Description: c.String("app-description"),
				Store:       c.String("app-store"),
				PocoVersion: pocoVersion(),
			},
		),
		bundler.WithDirectory(c.String("directory")),
	}

	compression := c.String("compression")
	if compression == "" {
		compression = "xz"
	}
	opts = append(opts, bundler.WithCompression(compression))

	b, err := bundler.New(opts...)
	if err != nil {
		pterm.Fatal.Println(err)
	}
	return b
}

func main() {

	app := &cli.App{
		Name:        "poco",
		Version:     pocoVersion(),
		Author:      "Ettore Di Giacinto",
		Usage:       "poco (bundle|render|pack|unpack)",
		Description: "poco bundles container images as portable static binaries",
		UsageText: `
Poco can build portable, statically linked binaries from containers.

For example:

$ CGO_ENABLED=0 poco bundle --image alpine --output alpine

will create an alpine binary which contains the alpine image and will start by default /bin/sh.

To try it, run: ./alpine

Every generated binary has a help too and take several options, use the --help on the generated binary.
		
`,
		Copyright: "Ettore Di Giacinto",

		Commands: []cli.Command{
			{
				Flags:     common(),
				Name:      "render",
				Aliases:   []string{"r"},
				UsageText: "poco render --image foo /dst",
				Usage:     "render golang code from container images",
				Description: `
Render golang generated files to the supplied dir
				
				$ poco render --image foo /dst
				`,
				Action: func(c *cli.Context) error {
					k := cliParse(c)
					if c.Args().First() == "" {
						return errors.New("need one parameter at least")
					}
					pterm.Info.Println("Rendering in", c.Args().First())
					k.Render(c.Args().First())
					return nil
				},
			},
			{
				Name:      "extract",
				Aliases:   []string{"e"},
				Usage:     "extract a binary and its libraries into a folder",
				UsageText: "extract <BIN> <DIR>",
				Action: func(c *cli.Context) error {
					src := c.Args()[0]
					dst := c.Args()[1]
					return extractor.Extract(extractor.WithFiles(src), extractor.WithOutputDir(dst))
				},
			},
			{
				Flags:     common(),
				Name:      "unpack",
				Aliases:   []string{"u"},
				Usage:     "unpacks a container image into a directory",
				UsageText: "unpack <IMAGE> <DIR>",
				Action: func(c *cli.Context) error {
					k := cliParse(c)
					src := c.Args()[0]
					dst := c.Args()[1]
					pterm.Info.Printfln(
						"Downloading image '%s' and unpacking into '%s' (local daemon: %t)",
						src, dst, c.Bool("local"),
					)

					return k.DownloadImage(src, dst, c.Bool("local"))
				},
			},
			{
				Name:    "pack",
				Aliases: []string{"p"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "destination",
						Usage: "Destination image tar",
						Value: "output.tar",
					},
					&cli.StringFlag{
						Name:  "os",
						Value: runtime.GOOS,
						Usage: "Overrides default image OS",
					},
					&cli.StringFlag{
						Name:  "arch",
						Value: runtime.GOARCH,
						Usage: "Overrides default image ARCH",
					},
				},
				Description: `
Packs files inside a tar which is consumable by docker.
E.g.
$ poco --destination out.tar foo/image:tar srcfile1 srcfile2 srcdir1 ...
$ docker load -i out.tar
$ docker push foo/image:tar ...
`,
				Usage: "pack a directory as a container image",
				Action: func(c *cli.Context) error {
					if !c.Args().Present() {
						return errors.New("need an image and source files to include inside the tar")
					}
					if c.Bool("debug") {
						pterm.EnableDebugMessages()
					}
					dst := c.String("destination")
					img := c.Args().First()
					src := c.Args().Tail()

					dir, err := os.MkdirTemp("", "containerbay")
					if err != nil {
						return err
					}
					defer os.RemoveAll(dir)

					err = archiver.Archive(src, filepath.Join(dir, "archive.tar"))
					if err != nil {
						return err
					}
					pterm.Info.Printfln("Creating '%s' as '%s' from %v", dst, img, src)
					return image.CreateTar(filepath.Join(dir, "archive.tar"), dst, img, c.String("arch"), c.String("os"))
				},
			},
			{
				Name: "pack-assets",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "C",
						Usage: "Change dir",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "compression",
						Usage: "Compression format",
						Value: "zst",
					},
					&cli.StringFlag{
						Name:  "destination",
						Usage: "Destination",
						Value: "assets",
					},
				},
				Description: `
				Packs files inside a tar which is consumable by docker.
				E.g.
				$ poco pack-assets srcfile1 srcfile2 srcdir1
				`,
				Usage: "pack files as assets",
				Action: func(c *cli.Context) error {
					if !c.Args().Present() {
						return errors.New("need an image and source files to include inside the tar")
					}
					if c.Bool("debug") {
						pterm.EnableDebugMessages()
					}
					dst := c.String("destination")
					src := c.Args()
					compressionType := c.String("compression")

					changeDir := c.String("C")

					pterm.Info.Printfln(
						"Creating '%s' from '%s', %s compression",
						dst,
						strings.Join(src, " "),
						compressionType,
					)

					if path.Ext(dst) == "" {
						dst = fmt.Sprintf("%s.tar.%s", dst, compressionType)
					} else {
						dst = fmt.Sprintf("%s.tar.%s", strings.ReplaceAll(dst, path.Ext(dst), ""), compressionType)
					}

					var cwd string
					if changeDir != "" {
						if !path.IsAbs(dst) {
							var err error
							cwd, err = os.Getwd()
							if err != nil {
								return err
							}
						}
						os.Chdir(changeDir)
					}

					err := archiver.Archive(src, dst)
					if err != nil {
						return err
					}

					if changeDir != "" {
						os.Chdir(cwd)
					}

					// This is to preserve '.' inside the resulting archive
					if !path.IsAbs(dst) && changeDir != "" {
						output := path.Join(changeDir, path.Base(dst))
						defer os.RemoveAll(output)
						return copy.Copy(output, path.Join(cwd, dst))
					}
					return nil
				},
			},
			{
				Flags: append(
					common(),
					&cli.StringFlag{
						Name:  "compression",
						Usage: "Compression format",
						Value: "zst",
					},
				),
				Name:      "bundle",
				Aliases:   []string{"b"},
				UsageText: "bundle --image <IMAGE> --entrypoint /bin/sh",
				Usage:     "generate golang binary from container images",
				Description: `Bundle containers into portable binaries

For example,

$ CGO_ENABLED=0 poco bundle --local --image kodi:latest --output kodi --entrypoint /usr/bin/kodi --app-mounts /sys --app-mounts /tmp --app-mounts /run --app-store '$HOME/.foo'

Creates a portable binary 'kodi' from the 'kodi:latest' image available in the local Docker daemon (--local).
It also associates to automatically mount /sys, /tmp and /run by default when starting and will unpack the binary content inside the user $HOME/.foo directory (be careful of the single quote).

				`,
				Action: func(c *cli.Context) (err error) {
					k := cliParse(c)

					pterm.Info.Printfln(
						"Creating bundle '%s' (version %s) from image '%s' with entrypoint '%s'",
						c.String("app-name"),
						c.String("app-version"),
						c.String("image"),
						c.String("entrypoint"),
					)

					mounts := c.StringSlice("app-mounts")
					if len(mounts) > 0 {
						pterm.Info.Printfln(
							"Default mounts: %s", strings.Join(mounts, " "),
						)
					}

					return k.Build(c.String("output"))
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
