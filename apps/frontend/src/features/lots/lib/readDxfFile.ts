import { DxfParseError, parseDxf } from './parseDxf'
import { isDxfFileName } from './isDxfFileName'
import type { DxfParseResult } from '../types'

const MAX_DXF_FILE_BYTES = 20_000_000

export async function readDxfFile(file: File): Promise<DxfParseResult> {
  if (!isDxfFileName(file.name)) {
    throw new DxfParseError('El archivo debe tener extensión .dxf.')
  }

  if (file.size > MAX_DXF_FILE_BYTES) {
    throw new DxfParseError('El archivo DXF supera el tamaño máximo permitido.')
  }

  const text = await file.text()
  return parseDxf(text)
}
