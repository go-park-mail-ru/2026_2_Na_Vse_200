SELECT t.id, t.title, m.duration_ms
FROM public.track t JOIN public.media_file m ON m.id = t.audio_file_id
WHERE t.status = 'published'
ORDER BY t.created_at DESC, t.id
LIMIT 20;
