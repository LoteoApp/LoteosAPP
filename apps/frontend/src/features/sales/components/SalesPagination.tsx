import { Button } from '../../../shared/ui/button'
import type { SalePage } from '../types'

type SalesPaginationProps = {
  page: Pick<SalePage, 'pagina' | 'paginas' | 'total'>
  isLoading: boolean
  onPageChange: (page: number) => void
}

export default function SalesPagination({ page, isLoading, onPageChange }: SalesPaginationProps) {
  if (page.paginas <= 1) {
    return null
  }

  return (
    <nav className="flex flex-wrap items-center justify-between gap-3" aria-label="Paginación de ventas">
      <p className="text-sm text-muted-foreground">
        Página {page.pagina} de {page.paginas} · {page.total} ventas
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={page.pagina <= 1 || isLoading}
          onClick={() => onPageChange(Math.max(1, page.pagina - 1))}
        >
          Anterior
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={page.pagina >= page.paginas || isLoading}
          onClick={() => onPageChange(Math.min(page.paginas, page.pagina + 1))}
        >
          Siguiente
        </Button>
      </div>
    </nav>
  )
}
