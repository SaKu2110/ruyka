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
    this.latestAnswer = document.getElementById('local-session-description-content');

    this.#newRTCPeerConnection();
    this.#updateStream();
  };

  connect() {
    const ws = new WebSocket("{{.}}");
    ws.onmessage = async (event) => {
      const message = JSON.parse(event.data);
      if (!message) return;

      try {
        switch (message.event) {
          case 'offer':
            const offer = message.sdp;
            if (!offer) return;

            await this.peer.setRemoteDescription(offer);
            this.#updateSenderTrack();

            const answer = await this.peer.createAnswer();
            await this.peer.setLocalDescription(answer);

            // sdpを画面上に表示する
            this.latestAnswer.innerText = answer.sdp;
            ws.send(JSON.stringify({ event: 'answer', sdp: answer }));
            return;
          case 'candidate':
            const candidate = message.ice;
            if (!candidate) return;

            await this.peer.addIceCandidate(candidate);
            return;
        };
      } catch (error) {
        window.alert(error);
      };
    };
    ws.onclose = () => { /* TODO: implement here */ };
    ws.onerror = () => { /* TODO: implement here */ };
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
    this.#updateStream();

    // ステータスバッチの表示処理
    const batchClassList = document.getElementById('status-batch').classList;
    if (batchClassList.contains('-checking')) batchClassList.remove('-checking');
    if (batchClassList.contains('-active')) batchClassList.remove('-active');
  };

  switchMediaType() {
    this.mediaType = (this.mediaType === 'video') ? 'display' : 'video';
    this.#updateStream();
  };

  #newRTCPeerConnection() {
    // FIXME: set ice servers parameter
    const peer = new RTCPeerConnection();
    peer.ontrack = (event) => {
      if (event.track.kind === 'audio') return;

      const video = document.createElement(event.track.kind);
      video.srcObject = event.streams[0];
      video.autoplay = true;
      video.disablePictureInPicture = true;
      this.remoteVideos.appendChild(video);

      event.track.onmute = () => video.play();
      event.streams[0].onremovetrack = () => {
        if (video.parentNode) video.parentNode.removeChild(video);
      };
    };
    peer.onicecandidate = (event) => {
      if (!event.cancelable) return;
      this.ws.send(JSON.stringify({
        event: 'candidate',
        ice: event.candidate,
      }));
    };
    peer.onconnectionstatechange = () => {
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

  // stream を張り替える
  async #updateStream() {
    try {
      const stream = (this.mediaType === 'video')
        ? await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
        : await navigator.mediaDevices.getDisplayMedia({ video: true, audio: true });
      this.stream = stream;
      this.localVideo.srcObject = stream;

      this.#updateSenderTrack();
    } catch (error) {
      window.alert(error);
    };
  };

  // Transceiver の track と stream を張り替える
  #updateSenderTrack() {
    const transceivers = this.peer.getTransceivers();
    if (transceivers.length === 0) return;

    this.stream.getAudioTracks().forEach(async track => {
      const transceiver = transceivers.find(tr => tr.mid === '0');
      if (!!transceiver) {
        transceiver.sender.replaceTrack(track);
        transceiver.sender.setStreams(this.stream);
        transceiver.direction = 'sendrecv';
      };
    });
    this.stream.getVideoTracks().forEach(async track => {
      const transceiver = transceivers.find(tr => tr.mid === '1');
      if (!!transceiver) {
        transceiver.sender.replaceTrack(track);
        transceiver.sender.setStreams(this.stream);
        transceiver.direction = 'sendrecv';
      };
    });
  };
};

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
  client.switchMediaType();
};
