declare module "@soundtouchjs/audio-worklet" {
  export class SoundTouchNode {
    static register(context: AudioContext, processorPath: string): Promise<void>;
    constructor(options: { context: AudioContext });
    readonly playbackRate: { value: number };
    readonly pitchSemitones: { value: number };
    readonly pitch: { value: number };
    connect(destination: AudioNode | AudioParam): AudioNode;
  }
}
