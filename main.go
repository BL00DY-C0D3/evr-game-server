// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main is a very simple server with UDP (default), TCP, or both
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	coresdk "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
	"github.com/samber/lo"
	"golang.org/x/time/rate"
)

var (
	build = "development"

	loggerFlags = log.LstdFlags | log.Ldate | log.Ltime | log.Lmsgprefix

	// This includes every flag that the game engine supports.
	// The help text has been improved, but the flags are the same.
	commandLine                      = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	HelpFlag                         = flag.Bool("help", false, "Display command line help information.")
	CrashFlag                        = flag.Bool("crash", false, "Force the application to crash.")
	CrashDeferredFlag                = flag.Bool("crashdeferred", false, "Force the application to crash, deferred to the first update loop.")
	LevelFlag                        = flag.String("level", "", "Specify a level to load.")
	StartLevelFlag                   = flag.String("startlevel", "", "Specify a starting level to load.")
	CheckpointFlag                   = flag.String("checkpoint", "", "Specify an active checkpoint to load.")
	StartCheckpointFlag              = flag.String("startcheckpoint", "", "Specify a starting checkpoint to load.")
	StartMissionEditorCheckpointFlag = flag.Bool("startmissioneditorcheckpoint", false, "Use checkpoint selected in Mission Script Editor.")
	NoLevelLoadsFlag                 = flag.Bool("nolevelloads", false, "Disable level loads.")
	NoAudioFlag                      = flag.Bool("noaudio", false, "Disable audio.")
	SpeakerSetupFlag                 = flag.String("speakersetup", "", "Specify speaker setup (stereo/5point1).")
	PanningRuleFlag                  = flag.String("panningrule", "", "Specify panning rule (speakers/headphones).")
	DefaultPortsFlag                 = flag.Bool("defaultports", false, "Disable dynamic ports for Wwise.")
	DataRootFlag                     = flag.String("dataroot", "", "Set an alternative data directory root (defaults to the project directory)")
	DataDirFlag                      = flag.String("datadir", "", "Set an alternative data directory to load data from")
	NoSymbolLookupFlag               = flag.Bool("nosymbollookup", false, "Disable symbol lookup")
	PackageNameFlag                  = flag.String("package", "", "Specify package name for resource loading")
	ReportCrashesFlag                = flag.Bool("reportcrashes", false, "Enable crash reporting")
	LegacyVisFlag                    = flag.Bool("legacyvis", false, "Use the legacy (precomputed) visibility system")
	CaptureVideoFlag                 = flag.Bool("capturevideo", false, "Enable video capture (also use -capture360)")
	CaptureAudioFlag                 = flag.Bool("captureaudio", false, "Enable audio capture")
	FullScreenFlag                   = flag.Bool("fullscreen", false, "Start the game in fullscreen mode")
	DisplayFlag                      = flag.Int("display", 0, "Specify the index of the display for fullscreen mode")
	AdapterFlag                      = flag.Int("adapter", 0, "Specify the index of the GPU to use")
	FullScreenResFlag                = flag.String("fullscreen_res", "", "Set the target fullscreen resolution")
	LanguageFlag                     = flag.String("language", "", "Set the start language")
	HeadlessFlag                     = flag.Bool("headless", false, "Run the game without graphics")
	FixedTimestepFlag                = flag.Bool("fixedtimestep", false, "Set the game to run at a fixed time step")
	SyncIntervalFlag                 = flag.Int("syncinterval", 0, "Set the VSYNC interval to use")
	NoMsaaFlag                       = flag.Bool("nomsaa", false, "Disable MSAA")
	VrScaleFlag                      = flag.Float64("vrscale", 0, "Scale VR resolution to improve performance")
	SmoothRotationFlag               = flag.Bool("smoothrotation", false, "Enable smooth rotation")
	SmoothRollFlag                   = flag.Bool("smoothroll", false, "Enable smooth roll")
	TestStartFlag                    = flag.Bool("teststart", false, "Start and stop the game to test for memory leaks")
	MsaaModeFlag                     = flag.String("msaa", "", "Specify the MSAA mode to use")
	UseTouchFlag                     = flag.Bool("usetouch", false, "Enable touch controls for the game")
	MpFlag                           = flag.Bool("mp", false, "Start in multiplayer mode")
	LogPathFlag                      = flag.String("logpath", "", "Specify the path for the log file")
	UniqueLogDirFlag                 = flag.Bool("uniquelogdir", false, "Write all log files to a unique directory")
	UserCfgPathFlag                  = flag.String("usercfgpath", "", "Specify the path for the local config")
	UseVrSizeFlag                    = flag.Bool("usevrsize", false, "Set the window size to VR native resolution")
	CaptureFlag                      = flag.Bool("capture", false, "Run the game in a mode designed for recording with a capture device")
	CaptureVp2Flag                   = flag.Bool("capturevp2", false, "Run the game in recording mode, using a second viewport.")
	DumpStatsFlag                    = flag.Bool("dumpstats", false, "Dump performance stats after 60 seconds and exit for automated performance collection")
	DefaultSettingsFlag              = flag.Bool("defaultsettings", false, "Force the game to auto-pick graphics settings")
	LobbyIdFlag                      = flag.String("lobbyid", "", "Specify the ID of the lobby to join on start")
	LobbyTeamFlag                    = flag.String("lobbyteam", "", "Specify the team to join on start")
	ModeratorFlag                    = flag.Bool("moderator", false, "Join the logged in user's lobby group as a moderator")
	ModerateUserFlag                 = flag.String("moderateuser", "", "Join the specified user's lobby as a moderator")
	ModerateGroupFlag                = flag.String("moderategroup", "", "Join the specified lobby group as a moderator")
	DisplayNameFlag                  = flag.String("displayname", "", "Set the logged in user's display name (if allowed)")
	MicProviderFlag                  = flag.String("micprovider", "", "Specify the desired microphone provider (OVR, RAD, DMO)")
	PortFlag                         = flag.Int("port", 0, "Specify the first port to try binding to (dedicated server only)")
	MpAppIdFlag                      = flag.String("mpappid", "", "Override multiplayer app id")
	SpectatorStreamFlag              = flag.Bool("spectatorstream", false, "Stream spectator mode matches")
	NumTaskThreadsFlag               = flag.Int("numtaskthreads", 0, "Change the number of task threads at startup")
	HttpPortFlag                     = flag.Int("httpport", 0, "Specify the port for the HTTP listener to use")
	NoOvrFlag                        = flag.Bool("noovr", false, "Disable OVR platform features")
	RegionFlag                       = flag.String("region", "", "Specify the region to use when searching for a server")
	GameTypeFlag                     = flag.String("gametype", "", "Specify the game type to create/find/join on start")
	PublisherLockFlag                = flag.String("publisherlock", "", "Set the publisher lock")
	ConfigHostFlag                   = flag.String("confighost", "", "Set the config host endpoint")
	LoginHostFlag                    = flag.String("loginhost", "", "Set the login host endpoint")
	RadServerDbHostFlag              = flag.String("radserverdbhost", "", "Set the server database host endpoint")
	ServerManagerHostFlag            = flag.String("servermanagerhost", "", "Set the server manager host endpoint")
	ServerCountryFlag                = flag.String("servercountry", "", "Set the server country")
	ServerLocationFlag               = flag.String("serverlocation", "", "Set the server location")
	ServerPluginFlag                 = flag.String("serverplugin", "", "Set the server plugin")
	ServerRegionFlag                 = flag.String("serverregion", "", "Set the server region")
	ServerFlag                       = flag.Bool("server", false, "[EchoRelay] Run as a dedicated game server")
	OfflineFlag                      = flag.Bool("offline", false, "[EchoRelay] Run the game in offline mode")
	WindowedFlag                     = flag.Bool("windowed", false, "[EchoRelay] Run the game without a headset, in a window")
	TimeStepFlag                     = flag.Int("timestep", 0, "[EchoRelay] Set the fixed update interval when using -headless (in ticks/updates per second). 0 = no fixed time step, 120 = default")
	ShutdownDelaySecFlag             = flag.Int("shutdowndelay", 0, "[wrapper] If greater than zero, automatically shut down the server this many seconds after the server becomes allocated")
	ReadyDelaySecFlag                = flag.Int("readydelay", 0, "[wrapper]If greater than zero, wait this many seconds each time before marking the game server as ready")
	ReadyIterationsFlag              = flag.Int("readyiterations", 0, "[wrapper] If greater than zero, return to a ready state this number of times before shutting down")
	GracefulTerminationDelaySecFlag  = flag.Int("gracefulterminationdelay", 0, "[wrapper]Delay before termination (by SIGKILL or automaticShutdownDelaySec)")
	InterceptLoginFlag               = flag.Bool("interceptlogin", false, "[wrapper] The port for the TCP proxy to listen for the login connection from EchoVR")
	VerboseFlag                      = flag.Bool("verbose", false, "[wrapper] Enable verbose logging")
)

func checkFlagCollisions() {
	if *DefaultSettingsFlag && *FullScreenResFlag != "" {
		fmt.Println("Warning: Cannot set fullscreen resolution with default settings.")
	}

	if *ModeratorFlag && (*ModerateUserFlag != "" || *ModerateGroupFlag != "") {
		fmt.Println("Warning: Cannot join as a moderator with a specified user or group.")
	}

	if *MpAppIdFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot override multiplayer app id in offline mode.")
	}

	if *SpectatorStreamFlag && *OfflineFlag {
		fmt.Println("Warning: Cannot stream spectator mode matches in offline mode.")
	}

	if *RegionFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot specify the region to use when searching for a server in offline mode.")
	}

	if *PublisherLockFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the publisher lock in offline mode.")
	}

	if *ConfigHostFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the config host endpoint in offline mode.")
	}

	if *LoginHostFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the login host endpoint in offline mode.")
	}

	if *RadServerDbHostFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the server database host endpoint in offline mode.")
	}

	if *ServerManagerHostFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the server manager host endpoint in offline mode.")
	}

	if *ServerCountryFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the server country in offline mode.")
	}

	if *ServerLocationFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the server location in offline mode.")
	}

	if *ServerRegionFlag != "" && *OfflineFlag {
		fmt.Println("Warning: Cannot set the server region in offline mode.")
	}

	if *ServerFlag && *OfflineFlag {
		fmt.Println("Warning: Cannot run as a dedicated game server in offline mode.")
	}

	if *ReadyIterationsFlag > 0 && *ShutdownDelaySecFlag <= 0 {
		log.Fatalf("Must set a shutdown delay if using ready iterations")
	}

}

// main starts a UDP or TCP server
func main() {
	sigCtx, _ := signals.NewSigKillContext()

	if argstr := os.Getenv("ECHOVR_ARGS"); argstr != "" {
		commandLine.Parse(strings.Fields(argstr))
	}

	commandLine.Parse(os.Args[1:])

	checkFlagCollisions()

	// Configure logging
	if *VerboseFlag {
		loggerFlags |= log.Llongfile
	} else {
		loggerFlags |= log.Lshortfile
	}

	log.SetFlags(loggerFlags)

	log.Printf("Starting wrapper v%s", build)

	// Check for incompatible flags.

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	ctx, cancel := context.WithCancel(context.Background())
	go doHealth(s, ctx)

	var gs *coresdk.GameServer
	gs, err = s.GameServer()
	if err != nil {
		log.Fatalf("Could not get gameserver port details: %s", err)
	}

	log.Printf("%s", gs.Status.Ports)
	if len(gs.Status.Ports) == 2 {
		HttpPortFlag = lo.ToPtr(int(gs.Status.Ports[0].Port))
		PortFlag = lo.ToPtr(int(gs.Status.Ports[1].Port))
	}

	apiProxy := NewAPICachingProxy(HttpPortFlag, PortFlag, lo.ToPtr(30))

	// Setup the login TCP Proxy
	// Get the origin address from the config

	config := LoadConfig("../../_local/config.json")
	loginAddress := config.GetLoginAddress()

	tcpProxyAddress := "127.0.0.1:16789"
	loginProxy := NewTCPProxy(tcpProxyAddress, loginAddress)
	if *InterceptLoginFlag {
		LoginHostFlag = lo.ToPtr(tcpProxyAddress)
		loginProxy.Start(ctx, cancel, apiProxy)
	}

	if *ShutdownDelaySecFlag > 0 {
		shutdownAfterNAllocations(s, *ReadyIterationsFlag, *ShutdownDelaySecFlag)
	}

	// Build the command line
	// look through the command line flags and exclude wrapper
	// flags that are not supported by the game engine

	args := make([]string, 0)

	commandLine.VisitAll(func(f *flag.Flag) {
		if strings.Contains(f.Usage, "[wrapper]") {
			return
		}
		if f.DefValue == f.Value.String() {
			return
		}
		args = append(args, f.Name, f.Value.String())
	})

	args = append(args, "-httpport", strconv.FormatInt(int64(*HttpPortFlag), 10))
	args = append(args, "-port", strconv.FormatInt(int64(*PortFlag), 10))

	log.Printf("Warning: timestep frequency is not set, tickrate will be unlimited.")

	engine := NewGameEngine("./echovr.exe", args)

	engine.Start()
	go apiProxy.ListenAndProxy(ctx)

	// TODO FIXME Wait until the healtcheck says the server is connect to Nakama

	if *ReadyDelaySecFlag > 0 {
		log.Printf("Waiting %d seconds before moving to ready", *ReadyDelaySecFlag)
		time.Sleep(time.Duration(*ReadyDelaySecFlag) * time.Second)
	}

	log.Print("Marking this server as ready")
	ready(s)

	<-sigCtx.Done()

	log.Printf("Waiting %d seconds before exiting", *GracefulTerminationDelaySecFlag)
	time.Sleep(time.Duration(*GracefulTerminationDelaySecFlag) * time.Second)
	os.Exit(0)
}

// UserConfig is a struct that represents the config.json.
type UserConfig struct {
	PublisherLock *string `json:"publisher_lock,omitempty"`
	APIURL        *string `json:"apiservice_host,omitempty"`
	Config        *string `json:"configservice_host,omitempty"`
	Login         *string `json:"loginservice_host,omitempty"`
	Matching      *string `json:"matchingservice_host,omitempty"`
	AIP           *string `json:"transactionservice_host,omitempty"`
	ServerDB      *string `json:"serverdb_host,omitempty"`
}

func NewUserConfig(pubLock, api, config, login, match, aip, db *string) *UserConfig {
	return &UserConfig{
		PublisherLock: pubLock,
		APIURL:        api,
		Config:        config,
		Login:         login,
		Matching:      match,
		AIP:           aip,
		ServerDB:      db,
	}
}

func LoadConfig(path string) *UserConfig {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Could not read user config: %v", err)
	}
	return UnmarshalUserConfig(b)
}
func (u *UserConfig) Save(path string) {
	if err := os.WriteFile(path, u.Marshal(), 0644); err != nil {
		log.Fatalf("Could not write user config: %v", err)
	}
}

func UnmarshalUserConfig(b []byte) *UserConfig {
	var u UserConfig
	if err := json.Unmarshal(b, &u); err != nil {
		log.Fatalf("Could not unmarshal user config: %v", err)
	}
	return &u
}
func (u *UserConfig) Marshal() []byte {
	b, err := json.Marshal(u)
	if err != nil {
		log.Fatalf("Could not marshal user config: %v", err)
	}
	return b
}

// GetLoginAddress returns the login service address (IP:Port)
func (u *UserConfig) GetLoginAddress() string {
	// Parse the loginservice URL
	p, err := url.Parse(*u.Login)
	if err != nil {
		log.Fatalf("Could not parse loginservice URL: %v", err)
	}

	if p.Port() != "" {
		return p.Host
	}

	switch p.Scheme {
	case "ws":
		fallthrough
	case "http":
		return fmt.Sprintf("%s:%d", p.Host, 80)
	case "wss":
		fallthrough
	case "https":
		return fmt.Sprintf("%s:%d", p.Host, 443)
	default:
		log.Fatalf("Unknown scheme on login URL: %s", p.Scheme)
		return "" // won't get here.
	}
}

// GameEngine is a struct that represents the game engine process.
type GameEngine struct {
	sync.RWMutex
	Command    *exec.Cmd
	BinaryPath string
	Arguments  []string
}

func NewGameEngine(path string, args []string) *GameEngine {
	gamelog := log.New(os.Stdout, "", loggerFlags)

	args = append([]string{path}, args...)

	cmd := exec.Command("/usr/bin/wine", args...)

	// Use a FIFO buffer to capture the game's log output.
	fifofn := fmt.Sprintf("%s/logfifo", os.TempDir())
	LogPathFlag = lo.ToPtr(fifofn)

	logs := make(chan string)

	// Capture the game's stderr output
	go func() {
		r, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Could not get stdout pipe: %v", err)
		}
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			logs <- scanner.Text()
		}
	}()

	// Capture the game's stdout output.
	go func() {
		r, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("Could not get stdout pipe: %v", err)
		}
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			logs <- scanner.Text()
		}
	}()

	// Capture the game's stderr output.
	go func() {
		if err := syscall.Mkfifo(fifofn, 0666); err != nil {
			log.Fatalf("failed to create fifo file: %v", err)
		}
		r, err := os.OpenFile(fifofn, os.O_RDONLY, 0)
		if err != nil {
			log.Fatalf("failed to open fifo file: %v", err)
		}
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			logs <- scanner.Text()
		}
	}()

	go func() {
		for log := range logs {
			filteredLog := filterLogMessage(log)
			if filteredLog != "" {
				gamelog.Println(filteredLog)
			}
		}
	}()

	return &GameEngine{
		Command: cmd,
	}
}

func filterLogMessage(s string) string {
	filters := []string{
		"has parent set to Auto",
	}
	for _, filter := range filters {
		if strings.Contains(s, filter) {
			return ""
		}
	}
	return s
}

func (e *GameEngine) Start() {
	e.Lock()
	log.Printf("Starting game engine with command: %s", strings.Join(e.Command.Args, " "))
	if err := e.Command.Start(); err != nil {
		log.Fatalf("Could not start game engine: %v", err)
	}
	e.Unlock()
	if err := e.Command.Wait(); err != nil {
		log.Fatalf("Game engine exited with error: %v", err)
	}
}

// Stop kills the game engine process, and the wrapper dies with it.
func (e *GameEngine) Stop() {
	e.Lock()
	defer e.Unlock()
	if e.Command != nil && e.Command.Process != nil {
		if err := e.Command.Process.Kill(); err != nil {
			log.Fatalf("Could not kill game engine: %v", err)
		}
	}
	log.Fatal("Game engine process has been killed")
}

// The TCPProxy is used to monitor the login connection. Once established,
// if the gameserver disconnects, the TCPProxy will close the connection and
// gracefully shutdown the GameServer process.
type TCPProxy struct {
	sync.RWMutex
	localAddress  string
	remoteAddress string
	listener      net.Listener
	log           *log.Logger
}

func NewTCPProxy(localAddress, remoteAddress string) *TCPProxy {
	return &TCPProxy{
		localAddress:  localAddress,
		remoteAddress: remoteAddress,
		log:           log.New(os.Stderr, "[loginproxy]", loggerFlags),
	}
}

func (p *TCPProxy) Start(ctx context.Context, cancel context.CancelFunc, apiCachingProxy *APICachingProxy) {
	var err error
	p.listener, err = net.Listen("tcp4", p.localAddress)
	if err != nil {
		p.log.Fatalf("Failed to listen on %s: %v", p.localAddress, err)
	}

	go func() {
		defer cancel()
		for {
			conn, err := p.listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			go p.handleConnection(ctx, conn)
		}
	}()
}

func (p *TCPProxy) handleConnection(ctx context.Context, localConn net.Conn) {
	remoteConn, err := net.Dial("tcp4", p.remoteAddress)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", p.remoteAddress, err)
	}
	// Shutdown the game engine if the connection dies

	go p.forward(localConn, remoteConn)
	go p.forward(remoteConn, localConn)

	<-ctx.Done()
	os.Exit(1)
}

func (p *TCPProxy) forward(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()
	// If the connection dies, gracefully shutdown the GameServer process

	io.Copy(src, dst)
}

type APICachingProxy struct {
	sync.RWMutex
	cachedData string
	originPort int
	listenPort int
	frequency  int
	logger     *log.Logger
	limiter    *rate.Limiter
	client     *http.Client
}

func NewAPICachingProxy(originPort, proxyPort, frequency *int) *APICachingProxy {

	return &APICachingProxy{
		cachedData: "",
		originPort: *originPort,
		listenPort: *proxyPort,
		frequency:  *frequency,
		logger:     log.New(os.Stderr, "[login ] ", loggerFlags),
		limiter:    rate.NewLimiter(rate.Limit(30), 1),
		client: &http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost: 1,
				MaxIdleConns:    1,
				IdleConnTimeout: 30 * time.Second,
			},
		},
	}
}

func (p *APICachingProxy) ListenAndProxy(ctx context.Context) {
	http.HandleFunc("/session", p.handleRequest)
	http.ListenAndServe(fmt.Sprintf(":%d", p.listenPort), nil)
}

// queryAPI queries the API and caches the response.
func (p *APICachingProxy) queryAPI(ctx context.Context) string {
	url := fmt.Sprintf("http://127.0.0.1:%d/session", p.originPort)

	// Create a rate limiter that will allow us to query the API up to 30 times a second.
	// If no client is connected, the rate limiter will block the query.

	// Query the API up to 30 times a second, as long as a client is connected.
	// If no client is connected, the rate limiter will block the query.
	resp, err := http.Get(url)
	if err != nil {
		p.logger.Println("Evr GET Error:", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Println("Evr Read Error:", err)
	}
	resp.Body.Close()
	return string(body)
}

func (p *APICachingProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	var data string
	if p.limiter.Allow() {
		// Allow the request even if the rate limit is exceeded
		ctx, _ := context.WithTimeout(r.Context(), 1000*time.Millisecond/30-3)
		p.Lock()
		data = p.queryAPI(ctx)
		p.cachedData = data
		p.Unlock()
	} else {
		p.RLock()
		data = p.cachedData
		p.RUnlock()
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, data)
}

// shutdownAfterNAllocations creates a callback to automatically shut down
// the server a specified number of seconds after the server becomes
// allocated the Nth time.
//
// The algorithm is:
//
//  1. Move the game server back to ready N times after it is allocated
//  2. Shutdown the game server after the Nth time is becomes allocated
//
// This follows the integration pattern documented on the website at
// https://agones.dev/site/docs/integration-patterns/reusing-gameservers/
func shutdownAfterNAllocations(s *sdk.SDK, readyIterations, shutdownDelaySec int) {
	gs, err := s.GameServer()
	if err != nil {
		log.Fatalf("Could not get game server: %v", err)
	}
	log.Printf("Initial game Server state = %s", gs.Status.State)

	m := sync.Mutex{} // protects the following two variables
	lastAllocated := gs.ObjectMeta.Annotations["agones.dev/last-allocated"]
	remainingIterations := readyIterations

	if err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		m.Lock()
		defer m.Unlock()
		la := gs.ObjectMeta.Annotations["agones.dev/last-allocated"]
		log.Printf("Watch Game Server callback fired. State = %s, Last Allocated = %q", gs.Status.State, la)
		if lastAllocated != la {
			log.Println("Game Server Allocated")
			lastAllocated = la
			remainingIterations--
			// Run asynchronously
			go func(iterations int) {
				time.Sleep(time.Duration(shutdownDelaySec) * time.Second)

				if iterations > 0 {
					log.Println("Moving Game Server back to Ready")
					readyErr := s.Ready()
					if readyErr != nil {
						log.Fatalf("Could not set game server to ready: %v", readyErr)
					}
					log.Println("Game Server is Ready")
					return
				}

				log.Println("Moving Game Server to Shutdown")
				if shutdownErr := s.Shutdown(); shutdownErr != nil {
					log.Fatalf("Could not shutdown game server: %v", shutdownErr)
				}
				// The process will exit when Agones removes the pod and the
				// container receives the SIGTERM signal
			}(remainingIterations)
		}
	}); err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

// ready attempts to mark this gameserver as ready
func ready(s *sdk.SDK) {
	err := s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, ctx context.Context) {
	tick := time.Tick(2 * time.Second)
	for {
		log.Printf("Health Ping")
		err := sdk.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		select {
		case <-ctx.Done():
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
