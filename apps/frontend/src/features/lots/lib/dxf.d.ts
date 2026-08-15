declare module 'dxf' {
  export class Helper {
    constructor(contents: string)
    readonly denormalised: unknown[]
    toPolylines(): { polylines: unknown[] }
  }
}
