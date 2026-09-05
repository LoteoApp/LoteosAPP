import { useEffect, useState, type ChangeEvent } from 'react'
import { Button } from '../../../shared/ui/button'
import { Input } from '../../../shared/ui/input'
import { Field, FieldLabel } from '../../../shared/ui/field'
import { ToggleGroup, ToggleGroupItem } from '../../../shared/ui/toggle-group'
import { fetchArchivoContent } from '../api/archivos'
import { useArchivos, type ArchivoTarget } from '../hooks/use-archivos'
import type { Archivo, ArchivoCategoria } from '../types'

const ARCHIVO_ACCEPT = 'image/jpeg,image/png,image/webp,application/pdf'

type ArchivosSectionProps = {
  target: ArchivoTarget
  accessToken: string | null
  canEdit: boolean
}

export default function ArchivosSection({ target, accessToken, canEdit }: ArchivosSectionProps) {
  const { archivos, isLoading, isSubmitting, error, upload, remove } = useArchivos(target, accessToken)
  const [categoria, setCategoria] = useState<ArchivoCategoria>('foto')

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    // Let the same file be picked again right after a failed upload.
    event.target.value = ''
    if (file) {
      await upload(categoria, file)
    }
  }

  return (
    <div className="flex flex-col gap-2 border-t border-border pt-3">
      <p className="text-xs font-medium text-muted-foreground">Fotos y planos</p>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      {canEdit && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Field orientation="horizontal" className="w-fit gap-2">
            <FieldLabel id={`archivo-categoria-${target.loteoId}`} className="sr-only">
              Tipo de archivo
            </FieldLabel>
            <ToggleGroup
              variant="outline"
              value={[categoria]}
              onValueChange={(next) => {
                const selected = next[0]
                if (selected === 'foto' || selected === 'plano') {
                  setCategoria(selected)
                }
              }}
              aria-labelledby={`archivo-categoria-${target.loteoId}`}
            >
              <ToggleGroupItem value="foto">Foto</ToggleGroupItem>
              <ToggleGroupItem value="plano">Plano</ToggleGroupItem>
            </ToggleGroup>
          </Field>
          <Input
            type="file"
            accept={ARCHIVO_ACCEPT}
            onChange={handleFileChange}
            disabled={isSubmitting}
            aria-label="Subir archivo"
          />
        </div>
      )}

      {isLoading && <p className="text-sm text-muted-foreground">Cargando archivos…</p>}
      {!isLoading && archivos.length === 0 && (
        <p className="text-sm text-muted-foreground">Todavía no hay archivos.</p>
      )}

      {archivos.length > 0 && (
        <ul className="flex flex-col gap-2">
          {archivos.map((archivo) => (
            <ArchivoListItem
              key={archivo.id}
              archivo={archivo}
              loteoId={target.loteoId}
              accessToken={accessToken}
              canEdit={canEdit}
              disabled={isSubmitting}
              onDelete={() => remove(archivo.id)}
            />
          ))}
        </ul>
      )}
    </div>
  )
}

type ArchivoListItemProps = {
  archivo: Archivo
  loteoId: string
  accessToken: string | null
  canEdit: boolean
  disabled: boolean
  onDelete: () => void
}

function ArchivoListItem({ archivo, loteoId, accessToken, canEdit, disabled, onDelete }: ArchivoListItemProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)

  useEffect(() => {
    if (!accessToken) {
      return
    }

    let cancelled = false
    let objectUrl: string | null = null

    fetchArchivoContent(loteoId, archivo.id, accessToken)
      .then((blob) => {
        if (cancelled) {
          return
        }
        objectUrl = URL.createObjectURL(blob)
        setPreviewUrl(objectUrl)
      })
      .catch(() => {
        // The row still shows the file name without a preview.
      })

    return () => {
      cancelled = true
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
      }
    }
  }, [loteoId, archivo.id, accessToken])

  const isImage = archivo.mimeType.startsWith('image/')

  return (
    <li className="flex items-center gap-3 rounded-md border border-border p-2">
      {isImage && previewUrl ? (
        <img
          src={previewUrl}
          alt={archivo.nombreOriginal}
          className="size-12 shrink-0 rounded object-cover"
        />
      ) : (
        <div className="flex size-12 shrink-0 items-center justify-center rounded bg-muted text-[0.65rem] font-medium text-muted-foreground">
          {archivo.categoria === 'plano' ? 'PDF' : 'IMG'}
        </div>
      )}

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm">{archivo.nombreOriginal}</p>
        <p className="text-xs text-muted-foreground">
          {archivo.categoria === 'foto' ? 'Foto' : 'Plano'}
        </p>
      </div>

      {previewUrl && (
        <a
          href={previewUrl}
          target="_blank"
          rel="noreferrer"
          className="shrink-0 text-xs text-muted-foreground underline"
        >
          Ver
        </a>
      )}

      {canEdit && (
        <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={onDelete}>
          Eliminar
        </Button>
      )}
    </li>
  )
}
