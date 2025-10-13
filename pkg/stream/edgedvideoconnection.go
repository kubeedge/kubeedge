package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"
)

type EdgedVideoConnection struct {
	ReadChan    chan *Message `json:"-"`
	Stop        chan struct{} `json:"-"`
	MessID      uint64        // message id
	URL         url.URL       `json:"url"`
	Header      http.Header   `json:"header"`
	ResourceUrl string        `json:"-"`
}

func (v *EdgedVideoConnection) GetMessageID() uint64 {
	return v.MessID
}

func (v *EdgedVideoConnection) CacheTunnelMessage(msg *Message) {
	v.ReadChan <- msg
}

func (v *EdgedVideoConnection) CloseReadChannel() {
	close(v.ReadChan)
}

func (v *EdgedVideoConnection) CleanChannel() {
	for {
		select {
		case <-v.Stop:
		default:
			return
		}
	}
}

func (v *EdgedVideoConnection) CreateConnectMessage() (*Message, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return NewMessage(v.MessID, MessageTypeVideoConnect, data), nil
}

func (v *EdgedVideoConnection) String() string {
	return fmt.Sprintf("EDGE_VIDEO_CONNECTOR Message MessageID %v", v.MessID)
}

func (v *EdgedVideoConnection) receiveFromCloudDataStream(stop chan struct{}) {
	for mess := range v.ReadChan {
		if mess.MessageType == MessageTypeRemoveConnect {
			klog.Infof("receive remove client id %v", mess.ConnectID)
			stop <- struct{}{}
		}
	}
	klog.V(6).Infof("%s read channel closed", v.String())
}

func (v *EdgedVideoConnection) write2CloudDataStream(tunnel SafeWriteTunneler, reader io.ReadCloser, stop chan struct{}) {
	defer func() {
		stop <- struct{}{}
	}()

	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if err != nil {
			klog.Errorf("[video] read video stream error: %v", err)
			return
		}

		msg := NewMessage(v.MessID, MessageTypeData, buf[:n])
		err = tunnel.WriteMessage(msg)

		if err != nil {
			if isClosedConnError(err) {
				klog.Warningf("[video] connection closed, stop writing: %v", err)
				return
			}
			klog.Warningf("[video] write tunnel message error: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

	}
}

func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}

	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure,
	) || websocket.IsUnexpectedCloseError(err)
}

func (v *EdgedVideoConnection) Serve(tunnel SafeWriteTunneler) error {
	if v.ResourceUrl == "" {
		return fmt.Errorf("resourceUrl is empty")
	}

	cmd := exec.Command(
		"ffmpeg",
		"-rtsp_transport", "tcp",
		"-i", v.ResourceUrl,
		"-r", "25",
		"-f", "mpegts",
		"-codec:v", "mpeg1video",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		klog.Errorf("create new video request error %v", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		klog.Errorf("Failed to start FFmpeg: %v", err)
		return err
	}
	defer cmd.Wait()
	defer func() {
		if cmd.Process != nil {
			// klog.Infof("[debug] Killing ffmpeg pid %d", cmd.Process.Pid)
			cmd.Process.Kill()
		}
	}()

	go v.receiveFromCloudDataStream(v.Stop)

	defer func() {
		for retry := 0; retry < 3; retry++ {
			msg := NewMessage(v.MessID, MessageTypeRemoveConnect, nil)
			if err := tunnel.WriteMessage(msg); err != nil {
				klog.Errorf("%v send %s message error %v", v, msg.MessageType, err)
			} else {
				break
			}
		}
	}()

	go v.write2CloudDataStream(tunnel, stdout, v.Stop)

	<-v.Stop
	klog.Infof("receive stop signal, so stop video scan ...")

	return nil
}

var _ EdgedConnection = &EdgedVideoConnection{}
