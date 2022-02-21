package nameflag

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/moby/term"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

const usageFmt = "Usage:\n  %s\n"

// it is not thread-safe.
type NameFlagSet struct {
	order    []string
	flagsets map[string]*pflag.FlagSet
}

func NewNameFlagSet() *NameFlagSet {
	return &NameFlagSet{order: make([]string, 0, 2), flagsets: make(map[string]*pflag.FlagSet)}
}

func (nfs *NameFlagSet) AddFlagSet(name string, fs *pflag.FlagSet) error {
	if nfs.flagsets[name] != nil {
		return fmt.Errorf("fs: %s already exist", name)
	}
	nfs.order = append(nfs.order, name)
	nfs.flagsets[name] = fs
	return nil
}

func (nfs *NameFlagSet) AddNameFlagSetToCmd(cmd *cobra.Command) {
	for _, fs := range nfs.flagsets {
		cmd.Flags().AddFlagSet(fs)
	}
}

func (nfs *NameFlagSet) SetUsageAndHelpFunc(cmd *cobra.Command) error {

	outFd, isTerminal := term.GetFdInfo(cmd.OutOrStdout())
	if !isTerminal {
		return fmt.Errorf("given writer is no terminal")
	}
	winsize, err := term.GetWinsize(outFd)
	if err != nil {
		return err
	}

	cols := int(winsize.Width)
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		printSections(cmd.OutOrStderr(), nfs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		printSections(cmd.OutOrStdout(), nfs, cols)
	})

	return nil
}

func printSections(w io.Writer, nfs *NameFlagSet, cols int) {
	for _, name := range nfs.order {
		fs := nfs.flagsets[name]
		if fs == nil {
			klog.Errorf("NameFlagSet:%s not exist", name)
			continue
		}
		if !fs.HasFlags() {
			continue
		}

		wideFS := pflag.NewFlagSet("", pflag.ExitOnError)
		wideFS.AddFlagSet(fs)

		var zzz string
		if cols > 24 {
			zzz = strings.Repeat("z", cols-24)
			wideFS.Int(zzz, 0, strings.Repeat("z", cols-24))
		}

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "\n%s flags:\n\n%s", strings.ToUpper(name[:1])+name[1:], wideFS.FlagUsagesWrapped(cols))

		if cols > 24 {
			i := strings.Index(buf.String(), zzz)
			lines := strings.Split(buf.String()[:i], "\n")
			fmt.Fprint(w, strings.Join(lines[:len(lines)-1], "\n"))
			fmt.Fprintln(w)
		} else {
			fmt.Fprint(w, buf.String())
		}
	}
}
