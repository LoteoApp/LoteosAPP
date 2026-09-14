import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import { Upload, X } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import { fetchAttachmentContent } from '../api/archivos'
import { useAttachments, type AttachmentTarget } from '../hooks/use-archivos'
import type { Attachment, AttachmentCategory } from '../types'

const ATTACHMENT_ACCEPT = 'image/jpeg,image/png,image/webp,application/pdf'

function categoryFor(file: File): AttachmentCategory {
  return file.type.startsWith('image/') ? 'foto' : 'plano'
}

type ArchivosSectionProps = {
  target: AttachmentTarget
  accessToken: string | null
  canEdit: boolean
  showLabel?: boolean
}

export default function ArchivosSection({
  target,
  accessToken,
  canEdit,
  showLabel = true,
}: ArchivosSectionProps) {
  const { attachments, isLoading, isSubmitting, error, upload, remove } = useAttachments(target, accessToken)
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    // Let the same file be picked again right after a failed upload.
    event.target.value = ''
    if (file) {
      await upload(categoryFor(file), file)
    }
  }

  return (
    <div className="flex flex-col gap-2">
      {showLabel && <p className="text-xs font-medium text-muted-foreground">Fotos y documentos</p>}

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      {canEdit && (
        <div>
          <Button
            type="button"
            variant="outline"
            disabled={isSubmitting}
            onClick={() => fileInputRef.current?.click()}
          >
            <Upload aria-hidden />
            Subir archivo
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept={ATTACHMENT_ACCEPT}
            onChange={handleFileChange}
            disabled={isSubmitting}
            aria-label="Subir archivo"
            className="hidden"
          />
        </div>
      )}

      {isLoading && <p className="text-sm text-muted-foreground">Cargando archivos…</p>}
      {!isLoading && attachments.length === 0 && (
        <p className="text-sm text-muted-foreground">Todavía no hay archivos.</p>
      )}

      {attachments.length > 0 && (
        // Fixed-height strip: with more attachments it scrolls sideways
        // instead of pushing the rest of the panel down.
        <ul className="flex gap-2 overflow-x-auto pb-1">
          {attachments.map((attachment) => (
            <AttachmentThumbnail
              key={attachment.id}
              attachment={attachment}
              loteoId={target.loteoId}
              accessToken={accessToken}
              canEdit={canEdit}
              disabled={isSubmitting}
              onDelete={() => remove(attachment.id)}
            />
          ))}
        </ul>
      )}
    </div>
  )
}

type AttachmentThumbnailProps = {
  attachment: Attachment
  loteoId: string
  accessToken: string | null
  canEdit: boolean
  disabled: boolean
  onDelete: () => void
}

function AttachmentThumbnail({
  attachment,
  loteoId,
  accessToken,
  canEdit,
  disabled,
  onDelete,
}: AttachmentThumbnailProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)

  useEffect(() => {
    if (!accessToken) {
      return
    }

    let cancelled = false
    let objectUrl: string | null = null

    fetchAttachmentContent(loteoId, attachment.id, accessToken)
      .then((blob) => {
        if (cancelled) {
          return
        }
        objectUrl = URL.createObjectURL(blob)
        setPreviewUrl(objectUrl)
      })
      .catch(() => {
        // The thumbnail still shows its fallback badge without a preview.
      })

    return () => {
      cancelled = true
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
      }
    }
  }, [loteoId, attachment.id, accessToken])

  const isImage = attachment.mimeType.startsWith('image/')

  const content = isImage && previewUrl ? (
    <img src={previewUrl} alt={attachment.nombreOriginal} className="size-full object-cover" />
  ) : (
    <div className="flex size-full items-center justify-center bg-muted text-[0.65rem] font-medium text-muted-foreground">
      {isImage ? 'IMG' : 'PDF'}
    </div>
  )

  return (
    <li className="relative size-20 shrink-0">
      {previewUrl ? (
        <a
          href={previewUrl}
          target="_blank"
          rel="noreferrer"
          title={attachment.nombreOriginal}
          className="block size-full overflow-hidden rounded-md border border-border"
        >
          {content}
        </a>
      ) : (
        <div title={attachment.nombreOriginal} className="size-full overflow-hidden rounded-md border border-border">
          {content}
        </div>
      )}

      {canEdit && (
        <Button
          type="button"
          variant="outline"
          size="icon-xs"
          disabled={disabled}
          onClick={onDelete}
          aria-label={`Eliminar ${attachment.nombreOriginal}`}
          className="absolute -top-1.5 -right-1.5 rounded-full bg-background"
        >
          <X aria-hidden />
        </Button>
      )}
    </li>
  )
}
