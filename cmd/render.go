package cmd

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kubernetix/k8x/v1/internal/ts"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// You generally won't need this unless you're processing stuff with
// complicated ANSI escape sequences. Turn it on if you notice flickering.
//
// Also keep in mind that high performance rendering only works for programs
// that use the full size of the terminal. We're enabling that below with
// tea.EnterAltScreen().
const useHighPerformanceRenderer = false

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type model struct {
	content  string
	ready    bool
	viewport viewport.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewport.SetContent(m.content)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m model) headerView() string {
	title := titleStyle.Render("k8x render view")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var (
	Interactive bool
)

func init() {
	render.PersistentFlags().BoolVarP(&Interactive, "interactive", "i", false, "Interactively show rendered json/yaml")
	rootCmd.AddCommand(render)
}

var namespaceTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    name: %s
`

var render = &cobra.Command{
	Use:   "render",
	Short: "Render a chart file as yaml or json",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(-1)
		}

		path := args[0]

		code := ts.Load(path, Verbose)
		export := ts.Run(code)

		content := []string{""}

		for _, component := range export["components"].([]interface{}) {
			if component == nil {
				continue
			}
			bts, _ := yaml.Marshal(component)
			content = append(content, string(bts))
		}

		namespace := export["namespace"]

		if Interactive {
			// Append auto generated namespace, Todo handle if namespace is undefined/null/""
			if hasValidNamespace(namespace) {
				content = append(content, fmt.Sprintf(namespaceTemplate, namespace, namespace))
			}

			md := fmt.Sprintf("```yml%s```", strings.Join(content, "---\n"))

			out, _ := glamour.Render(md, "dark")

			p := tea.NewProgram(
				model{content: out},
				tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
				tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
			)

			if _, err := p.Run(); err != nil {
				fmt.Println("could not run program:", err)
				os.Exit(1)
			}

			os.Exit(0)
		} else {
			// create and open a temporary file
			f, err := os.CreateTemp("", "tmpfile-") // in Go version older than 1.17 you can use ioutil.TempFile
			if err != nil {
				log.Fatal(err)
			}
			// close and remove the temporary file at the end of the program
			defer f.Close()
			defer os.Remove(f.Name())

			if hasValidNamespace(namespace) {
				// Writing the namespace template to the file, apply it, wait 5 seconds
				if _, err := f.Write([]byte(fmt.Sprintf(namespaceTemplate, export["namespace"], export["namespace"]))); err != nil {
					log.Fatal(err)
				}

				//fileOutput, _ := os.ReadFile(f.Name())
				//fmt.Println(string(fileOutput))

				grepCmd := exec.Command("kubectl", "apply", "-f", f.Name())

				output, _ := grepCmd.Output()
				fmt.Print(string(output))

				if strings.Contains(string(output), "created") {
					time.Sleep(1 * time.Second)
				}

				// Reset file
				err = f.Truncate(0)
				if err != nil {
					return
				}

				_, err = f.Seek(0, 0)
			}

			// Write chart
			if _, err := f.Write([]byte(strings.Join(content, "---\n"))); err != nil {
				log.Fatal(err)
			}

			//fileOutput, _ = os.ReadFile(f.Name())
			//fmt.Println(string(fileOutput))

			grepCmd := exec.Command("kubectl", "apply", "-f", f.Name())

			output, _ := grepCmd.Output()
			fmt.Println(string(output))
		}
	},
}

func hasValidNamespace(namespace interface{}) bool {
	if namespace == nil {
		return false
	}

	if namespace == "" {
		return false
	}

	return true
}
