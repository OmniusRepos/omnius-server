const API_BASE = '/admin/api';
const PUBLIC_API = '/api/v2';

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      ...options?.headers,
    },
  });

  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }

  return res.json();
}

// Stats
export async function getStats() {
  return request<{
    movies: number;
    series: number;
    channels: number;
    torrents: number;
  }>(`${API_BASE}/stats`);
}

// Movies
export interface Movie {
  id: number;
  imdb_code: string;
  title: string;
  title_english?: string;
  title_long?: string;
  slug?: string;
  year: number;
  rating: number;
  runtime?: number;
  genres: string[];
  summary?: string;
  description_full?: string;
  synopsis?: string;
  yt_trailer_code?: string;
  language?: string;
  mpa_rating?: string;
  background_image?: string;
  small_cover_image?: string;
  medium_cover_image?: string;
  large_cover_image?: string;
  imdb_rating?: number;
  imdb_votes?: string;
  rotten_tomatoes?: number;
  metacritic?: number;
  like_count?: number;
  download_count?: number;
  date_uploaded?: string;
  date_uploaded_unix?: number;
  content_type?: string;
  provider?: string;
  franchise?: string;
  torrents: Torrent[];
  cast?: Cast[];
}

export interface Cast {
  name: string;
  character_name: string;
  url_small_image?: string;
  imdb_code?: string;
}

export interface Torrent {
  id: number;
  hash: string;
  quality: string;
  type: string;
  size: string;
  seeds: number;
  peers: number;
}

export async function getMovies(params?: { page?: number; limit?: number; search?: string; sort_by?: string; order_by?: string }) {
  const query = new URLSearchParams();
  if (params?.page) query.set('page', String(params.page));
  if (params?.limit) query.set('limit', String(params.limit));
  if (params?.search) query.set('query_term', params.search);
  // Default to newest first
  query.set('sort_by', params?.sort_by || 'date_uploaded');
  query.set('order_by', params?.order_by || 'desc');

  const res = await request<{ status: string; data: { movies: Movie[]; movie_count: number } }>(
    `${PUBLIC_API}/list_movies.json?${query}`
  );
  return { movies: res.data.movies || [], total: res.data.movie_count };
}

export async function getMovie(id: number) {
  const res = await request<{ status: string; data: { movie: Movie } }>(
    `${PUBLIC_API}/movie_details.json?movie_id=${id}`
  );
  return res.data.movie;
}

export async function deleteMovie(id: number) {
  return request(`${API_BASE}/movies/${id}`, { method: 'DELETE' });
}

export async function getMovieByIMDB(imdbCode: string) {
  return request<{ exists: boolean; movie?: Movie }>(`${API_BASE}/movies/by-imdb/${imdbCode}`);
}

export async function updateMovie(id: number, data: Partial<{
  imdb_code: string;
  title: string;
  year: number;
  rating: number;
  runtime: number;
  genres: string;
  language: string;
  summary: string;
  yt_trailer_code: string;
  medium_cover_image: string;
  background_image: string;
}>) {
  return request<Movie>(`${API_BASE}/movies/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function addTorrent(movieId: number, data: { hash: string; quality: string; type: string; size?: string }) {
  return request(`${API_BASE}/movies/${movieId}/torrent`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// Series
export interface Series {
  id: number;
  imdb_code: string;
  title: string;
  year: number;
  rating: number;
  total_seasons: number;
  status: string;
  poster_image?: string;
}

export async function getSeries(params?: { page?: number; limit?: number; search?: string }) {
  const query = new URLSearchParams();
  if (params?.page) query.set('page', String(params.page));
  if (params?.limit) query.set('limit', String(params.limit));
  if (params?.search) query.set('query_term', params.search);

  const res = await request<{ status: string; data: { series: Series[]; series_count: number } }>(
    `${PUBLIC_API}/list_series.json?${query}`
  );
  return { series: res.data.series || [], total: res.data.series_count };
}

// Curated Lists
export interface CuratedList {
  id: number;
  name: string;
  slug: string;
  description?: string;
  sort_by: string;
  order_by: string;
  minimum_rating?: number;
  maximum_rating?: number;
  minimum_year?: number;
  maximum_year?: number;
  genre?: string;
  limit: number;
  is_active: boolean;
  display_order: number;
  movies?: Movie[];
}

export async function getCuratedLists() {
  return request<CuratedList[]>(`${API_BASE}/curated`);
}

export async function getCuratedList(id: number) {
  const res = await request<{ status: string; data: { list: CuratedList } }>(
    `${PUBLIC_API}/curated_list.json?list_id=${id}`
  );
  return res.data.list;
}

export async function createCuratedList(data: Partial<CuratedList>) {
  return request<CuratedList>(`${API_BASE}/curated`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateCuratedList(id: number, data: Partial<CuratedList>) {
  return request<CuratedList>(`${API_BASE}/curated/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteCuratedList(id: number) {
  return request(`${API_BASE}/curated/${id}`, { method: 'DELETE' });
}

export async function addMovieToList(listId: number, movieId: number, order?: number) {
  return request(`${API_BASE}/curated/${listId}/movies`, {
    method: 'POST',
    body: JSON.stringify({ movie_id: movieId, order: order || 0 }),
  });
}

export async function removeMovieFromList(listId: number, movieId: number) {
  return request(`${API_BASE}/curated/${listId}/movies/${movieId}`, { method: 'DELETE' });
}

// Auth
export async function logout() {
  window.location.href = '/admin/logout';
}
