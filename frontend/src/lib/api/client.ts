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
  // Rich data from IMDB
  director?: string;
  writers?: string[];
  budget?: string;
  box_office_gross?: string;
  country?: string;
  awards?: string;
  all_images?: string[];
  // Coming soon status
  status?: string;        // "available" or "coming_soon"
  release_date?: string;  // YYYY-MM-DD format
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
  title_slug?: string;
  year: number;
  rating: number;
  runtime?: number;
  genres?: string[];
  summary?: string;
  total_seasons: number;
  total_episodes?: number;
  status: string;
  network?: string;
  poster_image?: string;
  background_image?: string;
  date_added?: string;
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

export interface Episode {
  id: number;
  series_id: number;
  season_number: number;
  episode_number: number;
  title: string;
  summary?: string;
  air_date?: string;
  runtime?: number;
  still_image?: string;
  torrents?: EpisodeTorrent[];
}

export interface EpisodeTorrent {
  id: number;
  episode_id: number;
  hash: string;
  quality: string;
  seeds: number;
  peers: number;
  size: string;
}

export interface SeasonPack {
  id: number;
  series_id: number;
  season: number;
  hash: string;
  quality: string;
  seeds: number;
  peers: number;
  size: string;
  size_bytes: number;
}

export interface SeriesDetails extends Series {
  episodes?: Episode[];
  season_packs?: SeasonPack[];
}

export async function getSeriesDetails(id: number, withEpisodes = true): Promise<SeriesDetails> {
  const res = await request<{ status: string; data: { series: Series; episodes: Episode[]; season_packs: SeasonPack[] } }>(
    `${PUBLIC_API}/series_details.json?series_id=${id}&with_episodes=${withEpisodes}`
  );
  return {
    ...res.data.series,
    episodes: res.data.episodes || [],
    season_packs: res.data.season_packs || [],
  };
}

export async function getSeriesByIMDB(imdbCode: string) {
  return request<{ exists: boolean; series?: Series }>(`${API_BASE}/series/by-imdb/${imdbCode}`);
}

export async function deleteSeries(id: number) {
  return request(`${API_BASE}/series/${id}`, { method: 'DELETE' });
}

export async function updateSeries(id: number, data: Partial<{
  imdb_code: string;
  title: string;
  year: number;
  rating: number;
  runtime: number;
  genres: string;
  summary: string;
  poster_image: string;
  background_image: string;
  total_seasons: number;
  status: string;
  network: string;
}>) {
  return request<Series>(`${API_BASE}/series/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
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

// Channels (IPTV)
export interface Channel {
  id: string;
  name: string;
  country?: string;
  languages?: string[];
  categories?: string[];
  logo?: string;
  stream_url?: string;
  is_nsfw?: boolean;
  website?: string;
}

export interface ChannelCountry {
  code: string;
  name: string;
  flag?: string;
  channel_count?: number;
}

export interface ChannelCategory {
  id: string;
  name: string;
  channel_count?: number;
}

export interface ChannelEPG {
  id: number;
  channel_id: string;
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
}

export async function getChannels(params?: { page?: number; limit?: number; country?: string; category?: string; query_term?: string }) {
  const query = new URLSearchParams();
  if (params?.page) query.set('page', String(params.page));
  if (params?.limit) query.set('limit', String(params.limit));
  if (params?.country) query.set('country', params.country);
  if (params?.category) query.set('category', params.category);
  if (params?.query_term) query.set('query_term', params.query_term);

  const res = await request<{ status: string; data: { channels: Channel[]; channel_count: number; limit: number; page_number: number } }>(
    `${PUBLIC_API}/list_channels.json?${query}`
  );
  return { channels: res.data.channels || [], total: res.data.channel_count, limit: res.data.limit, page: res.data.page_number };
}

export async function getChannelCountries() {
  const res = await request<{ status: string; data: { countries: ChannelCountry[] } }>(
    `${PUBLIC_API}/channel_countries.json`
  );
  return res.data.countries || [];
}

export async function getChannelCategories() {
  const res = await request<{ status: string; data: { categories: ChannelCategory[] } }>(
    `${PUBLIC_API}/channel_categories.json`
  );
  return res.data.categories || [];
}

export async function getChannelEPG(channelId: string) {
  const res = await request<{ status: string; data: { epg: ChannelEPG[] } }>(
    `${PUBLIC_API}/channel_epg.json?channel_id=${channelId}`
  );
  return res.data.epg || [];
}

export async function syncIPTVChannels(m3uUrl?: string) {
  return request<{ status: string; message: string; m3u_url: string }>(`${API_BASE}/channels/sync`, {
    method: 'POST',
    body: JSON.stringify({ m3u_url: m3uUrl || '' }),
  });
}

export async function getIPTVSyncStatus() {
  return request<{
    running: boolean;
    phase: string;
    progress: number;
    total: number;
    last_sync?: string;
    last_error?: string;
    channels: number;
    countries: number;
    categories: number;
    m3u_url: string;
  }>(`${API_BASE}/channels/sync/status`);
}

export async function getChannelStats() {
  return request<{ channels: number; countries: number; categories: number; with_streams: number }>(
    `${API_BASE}/channels/stats`
  );
}

export async function getChannelSettings() {
  return request<{ m3u_url: string }>(`${API_BASE}/channels/settings`);
}

export async function updateChannelSettings(m3uUrl: string) {
  return request<{ status: string; m3u_url: string }>(`${API_BASE}/channels/settings`, {
    method: 'PUT',
    body: JSON.stringify({ m3u_url: m3uUrl }),
  });
}

export async function deleteChannel(id: string) {
  return request(`${API_BASE}/channels/${id}`, { method: 'DELETE' });
}

export async function startHealthCheck() {
  return request<{ status: string; message: string }>(`${API_BASE}/channels/health-check`, {
    method: 'POST',
  });
}

export async function getHealthCheckStatus() {
  return request<{
    running: boolean;
    phase: string;
    total: number;
    checked: number;
    removed: number;
    started_at?: string;
    completed_at?: string;
    last_error?: string;
  }>(`${API_BASE}/channels/health-check/status`);
}

export async function clearBlocklist() {
  return request<{ status: string; message: string }>(`${API_BASE}/channels/blocklist`, {
    method: 'DELETE',
  });
}

// Server Services Config
export interface ServiceConfig {
  id: string;
  label: string;
  enabled: boolean;
  icon: string;
  display_order: number;
}

export async function getServices() {
  return request<ServiceConfig[]>(`${API_BASE}/services`);
}

export async function updateServices(services: ServiceConfig[]) {
  return request<{ status: string }>(`${API_BASE}/services`, {
    method: 'PUT',
    body: JSON.stringify(services),
  });
}

// Subtitles
export interface StoredSubtitle {
  id: number;
  imdb_code: string;
  language: string;
  language_name: string;
  release_name: string;
  hearing_impaired: boolean;
  source: string;
  season_number?: number;
  episode_number?: number;
  created_at?: string;
}

export interface SubtitlePreview {
  id: number;
  preview: string;
  language: string;
  release_name: string;
  total_lines: number;
}

export async function getSubtitles(imdbCode: string) {
  return request<{ subtitles: StoredSubtitle[]; count: number }>(`${API_BASE}/subtitles?imdb_code=${encodeURIComponent(imdbCode)}`);
}

export async function getSubtitlePreview(id: number) {
  return request<SubtitlePreview>(`${API_BASE}/subtitles/${id}/preview`);
}

export async function deleteSubtitle(id: number) {
  return request<{ status: string }>(`${API_BASE}/subtitles/${id}`, { method: 'DELETE' });
}

export async function syncSubtitles(imdbCode: string, languages = 'en,sq,es,fr,de,it,pt,tr,ar,zh,ja,ko,ru', season?: number, episode?: number) {
  const body: Record<string, unknown> = { imdb_code: imdbCode, languages };
  if (season && episode) {
    body.season = season;
    body.episode = episode;
  }
  return request<{ status: string; stored: number; message: string }>(`${API_BASE}/subtitles/sync`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

// License (no longer used — license management is in Licenses.svelte directly)

export async function getLicenseStatus() {
  return request<LicenseStatus>(`${API_BASE}/license-status`);
}

// Auth
export async function logout() {
  window.location.href = '/admin/logout';
}
