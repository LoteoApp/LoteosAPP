import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Link } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { SearchField } from '../../../shared/ui/search-field'
import { useLoteos } from '../hooks/use-loteos'
import LoteoZocaloCard from '../components/LoteoZocaloCard'

type LoteosListPageProps = {
  accessToken: string | null
}

export default function LoteosListPage({ accessToken }: LoteosListPageProps) {
  const [search, setSearch] = useState('')
  const { loteos, isLoading, error } = useLoteos(accessToken ?? '', search)

  const isSearching = search.trim() !== ''
  // Once a load brings loteos the search box stays: clearing a search that
  // matched nothing leaves the list empty until the debounced refetch lands,
  // and unmounting the input there would steal the focus mid-typing.
  const [hasLoadedLoteos, setHasLoadedLoteos] = useState(false)
  if (!hasLoadedLoteos && loteos.length > 0) {
    setHasLoadedLoteos(true)
  }
  const showSearch = hasLoadedLoteos || isSearching

  return (
    <section className="flex flex-col gap-4">
      <header className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-semibold">Loteos</h1>
            <span className="text-sm text-muted-foreground">{loteos.length}</span>
          </div>
          <p className="text-sm text-muted-foreground">
            El listado de loteos cargados. Tocá uno para ver el plano y los lotes.
          </p>
        </div>
        <Button render={<Link to="/lotes/nuevo" />}>
          <Plus aria-hidden />
          Nuevo loteo
        </Button>
      </header>

      {showSearch && (
        <SearchField
          id="buscar-loteo"
          placeholder="Nombre o ubicación"
          value={search}
          onChange={setSearch}
          inputClassName="sm:max-w-xs"
        />
      )}

      {isLoading && loteos.length === 0 && (
        <p className="text-muted-foreground">Cargando loteos…</p>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo cargar el listado</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {!isLoading && !error && loteos.length === 0 && !isSearching && (
        <p className="text-muted-foreground">Todavía no hay loteos cargados.</p>
      )}

      {!isLoading && !error && loteos.length === 0 && isSearching && (
        <p className="text-muted-foreground">No se encontraron loteos con esa búsqueda.</p>
      )}

      {loteos.length > 0 && (
        <ul className="flex flex-col gap-3">
          {loteos.map((loteo) => (
            <li key={loteo.id}>
              <LoteoZocaloCard loteo={loteo} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
