
class RuykaClient {
  static trackKindTypeAudio = 'audio';
  static trackKindTypeVideo = 'video';

  constructor() {
    this.mediaType = 'video';

    // setup video elements
    const localVideo = document.createElement('video');
    localVideo.autoplay = true;
    localVideo.muted = true;
    document.getElementById('local-video').appendChild(localVideo);

    this.localVideo = localVideo;
    this.remoteVideos = document.getElementById('remote-videos');

    this.#newRTCPeerConnection();
    this.#setupLocalVideo();
  };

  connect() {
    const ws = new WebSocket("{{.}}");
    ws.onmessage = async (ev) => {
      const msg = JSON.parse(ev.data);
      if (!msg) return;

      try {
        switch (msg.event) {
          case 'offer':
            const offer = msg.sdp;
            if (offer) {
              await this.peer.setRemoteDescription(offer);
              const answer = await this.peer.createAnswer();
              await this.peer.setLocalDescription(answer);

              // sdpを画面上に表示する
              document.getElementById('local-session-description-content').innerText = answer.sdp;
              this.ws.send(JSON.stringify({
                event: 'answer',
                sdp: answer,
              }));
            }
            return;
          case 'candidate':
            const candidate = msg.ice;
            if (candidate) await this.peer.addIceCandidate(candidate);
            return;
        }
      } catch (err) {
        window.alert(err);
      };
    };
    ws.onclose = (_ev) => {
      // TODO: implement here.
    };
    ws.onerror = (_ev) => {
      // TODO: implement here.
    };
    this.ws = ws;
  };

  close() {
    this.peer.close();
    this.ws.close(1000);

    this.mediaType = 'video';
    this.remoteVideos.childNodes.forEach(node => {
      this.remoteVideos.removeChild(node);
    });
    this.#newRTCPeerConnection();
    this.#setupLocalVideo();

    // ステータスバッチの表示処理
    const batchClassList = document.getElementById('status-batch').classList;
    if (batchClassList.contains('-checking')) batchClassList.remove('-checking');
    if (batchClassList.contains('-active')) batchClassList.remove('-active');
  };

  switchVideoSource() {
    if (this.mediaType === 'video') {
      this.mediaType = 'display';
      this.#setupLocalVideo();
    } else {
      this.mediaType = 'video';
      this.#setupLocalVideo();
    }
  };

  // private
  #newRTCPeerConnection() {
    // FIXME: set ice servers parameter
    const peer = new RTCPeerConnection();
    peer.ontrack = (ev) => {
      if (ev.track.kind === 'audio') return;

      const video = document.createElement(ev.track.kind);
      video.srcObject = ev.streams[0];
      video.autoplay = true;
      video.disablePictureInPicture = true;
      this.remoteVideos.appendChild(video);

      ev.track.onmute = () => video.play();
      ev.streams[0].onremovetrack = () => {
        if (video.parentNode) video.parentNode.removeChild(video);
      };
    };
    peer.onicecandidate = (ev) => {
      if (!ev.cancelable) return;
      this.ws.send(JSON.stringify({
        event: 'candidate',
        ice: ev.candidate,
      }));
    };
    peer.onconnectionstatechange = (_ev) => {
      const batchClassList = document.getElementById('status-batch').classList;
      switch (this.peer.connectionState) {
        case 'checking':
        case 'connecting':
          batchClassList.add('-checking');
          break;
        case 'connected':
          if (batchClassList.contains('-checking')) batchClassList.remove('-checking');
          batchClassList.add('-active');
          break;
        default:
          if (batchClassList.contains('-checking')) batchClassList.remove('-checking');
          if (batchClassList.contains('-active')) batchClassList.remove('-active');
          break;
      }
    };
    this.peer = peer;
  };

  async #setupLocalVideo() {
    try {
      const stream = (this.mediaType === 'video')
        ? await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
        : await navigator.mediaDevices.getDisplayMedia({ video: true, audio: true });
      stream.getAudioTracks().forEach(async tr => {
        const sender = this.peer.getSenders().find(s => s.track.kind === tr.kind);
        if (sender) {
          await sender.replaceTrack(tr);
        } else {
          this.peer.addTrack(tr, stream);
        };
      });
      stream.getVideoTracks().forEach(async tr => {
        const sender = this.peer.getSenders().find(s => s.track.kind === tr.kind);
        if (sender) {
          await sender.replaceTrack(tr);
        } else {
          this.peer.addTrack(tr, stream);
          this.peer.addTransceiver(tr, {
            streams: [stream],
          });
        };
      });
      this.localVideo.srcObject = stream;
    } catch (err) {
      window.alert(err);
    }
  };
}

const client = new RuykaClient();
const connectButton = document.getElementById('connect-button');
const closeButton = document.getElementById('close-button');
const videoSourceToggleSwitch = document.getElementById('video-source-toggle-switch');

connectButton.onclick = () => {
  client.connect();
  connectButton.disabled = true;
  closeButton.disabled = false;
};
closeButton.onclick = () => {
  client.close();
  connectButton.disabled = false;
  closeButton.disabled = true;
};
videoSourceToggleSwitch.onclick = () => {
  client.switchVideoSource();
};
