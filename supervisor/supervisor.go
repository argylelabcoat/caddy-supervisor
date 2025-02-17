package supervisor

import (
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	maxRestartDelay             = 5 * time.Minute
	minRestartDelay             = 10 * time.Second
	durationToResetRestartDelay = 10 * time.Minute
)

// Supervisor provides functionality to start and supervise a background process
type Supervisor struct {
	options     *Options
	cmd         *exec.Cmd
	keepRunning bool
}

// CreateSupervisors creates a new process supervisor
func CreateSupervisors(options *Options) []*Supervisor {
	var supervisors []*Supervisor

	for i := 0; i < options.Replicas; i++ {
		templateData := &TemplateData{
			Replica: i,
		}

		replicaOptions := options.processTemplates(templateData)

		supervisor := &Supervisor{
			options: replicaOptions,
		}

		supervisors = append(supervisors, supervisor)
	}

	return supervisors
}

// Start a process and supervise
func (s *Supervisor) Start() {
	s.keepRunning = true
	go s.supervise()
}

func (s *Supervisor) supervise() {
	restartDelay := minRestartDelay

	for s.keepRunning {
		s.cmd = exec.Command(s.options.Command, s.options.Args...)

		s.cmd.Env = append(os.Environ(), s.options.Env...)

		if s.options.Dir != "" {
			s.cmd.Dir = s.options.Dir
		}

		if stdoutFile := getFile(s.options.RedirectStdout); stdoutFile != nil {
			s.cmd.Stdout = stdoutFile
			defer stdoutFile.Close()
		}

		if stderrFile := getFile(s.options.RedirectStderr); stderrFile != nil {
			s.cmd.Stderr = stderrFile
			defer stderrFile.Close()
		}

		start := time.Now()
		err := s.cmd.Run()
		duration := time.Now().Sub(start)

		if err != nil {
			log.Printf("Process error: %v\n", err)
		} else {
			log.Printf("Process exited after: %v\n", duration)
		}

		if !s.keepRunning {
			break
		}

		switch s.options.RestartPolicy {
		case RestartAlways:
			break
		case RestartOnFailure:
			if err == nil {
				s.keepRunning = false
			}
			break
		case RestartNever:
			s.keepRunning = false
			break
		}

		if s.keepRunning {
			if restartDelay > minRestartDelay && (err == nil || duration > durationToResetRestartDelay) {
				log.Printf("Resetting restart delay to %v\n", minRestartDelay)
				restartDelay = minRestartDelay
			}

			if err != nil {
				log.Printf("Restarting in %v\n", restartDelay)
				time.Sleep(restartDelay)
				restartDelay = increaseRestartDelay(restartDelay)
			}
		}
	}
}

// Stop the supervised process
func (s *Supervisor) Stop() {
	s.keepRunning = false

	if cmdIsRunning(s.cmd) {
		err := s.cmd.Process.Signal(os.Interrupt)
		if err == nil {
			go func() {
				time.Sleep(s.options.TerminationGracePeriod)
				if cmdIsRunning(s.cmd) {
					s.cmd.Process.Kill()
				}
			}()
			s.cmd.Process.Wait()
		} else {
			s.cmd.Process.Kill()
		}
	}
}

func cmdIsRunning(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.Process != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited())
}

func getFile(value string) *os.File {
	if value == "" {
		return nil
	}

	switch value {
	case "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		outFile, err := os.OpenFile(value, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			return nil
		}
		return outFile
	}
}

func increaseRestartDelay(restartDelay time.Duration) time.Duration {
	restartDelay = restartDelay * 2

	if restartDelay > maxRestartDelay {
		restartDelay = maxRestartDelay
	}

	return restartDelay
}
