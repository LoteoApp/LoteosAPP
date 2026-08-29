import type { ChangeEvent } from 'react'
import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { DxfParseError } from '../lib/parseDxf'
import { readDxfFile } from '../lib/readDxfFile'
import type { DxfParseResult } from '../types'

type DxfFileFieldProps = {
  fileName: string | null
  onParsed: (result: DxfParseResult, fileName: string) => void
  onError: (message: string) => void
  onCleared: () => void
}

export default function DxfFileField({
  fileName,
  onParsed,
  onError,
  onCleared,
}: DxfFileFieldProps) {
  async function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      onCleared()
      return
    }

    try {
      const result = await readDxfFile(file)
      onParsed(result, file.name)
    } catch (error) {
      const message =
        error instanceof DxfParseError
          ? error.message
          : 'No se pudo interpretar el archivo DXF.'
      onError(message)
    }
  }

  return (
    <Field>
      <FieldLabel htmlFor="loteo-dxf">Archivo DXF</FieldLabel>
      <Input
        id="loteo-dxf"
        name="dxf"
        type="file"
        accept=".dxf,application/dxf,image/vnd.dxf"
        onChange={handleChange}
      />
      <FieldDescription>
        {fileName
          ? fileName
          : 'Opcional. Capas LOTEO, MANZANA, LOTES y CALLE.'}
      </FieldDescription>
    </Field>
  )
}
