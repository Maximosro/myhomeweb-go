-- Dashboard "General" (default)
INSERT OR IGNORE INTO dashboards (id, name, display_order) VALUES ('d0000000-0000-0000-0000-000000000001', 'General', 1);

-- Categories
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000001', 'Streaming',       '🎬', 1, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000002', 'Compras',         '🛒', 2, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000003', 'Social',          '💬', 3, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000004', 'Tiendas de Juegos','🎮', 4, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000005', 'Gaming',          '🕹', 5, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000006', 'World of Warcraft','⚔', 6, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000007', 'Viajes',          '✈', 7, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000008', 'Inversiones',     '💰', 8, 'd0000000-0000-0000-0000-000000000001');
INSERT OR IGNORE INTO categories (id, name, icon, display_order, dashboard_id) VALUES ('a0000000-0000-0000-0000-000000000009', 'Banca',           '🏦', 9, 'd0000000-0000-0000-0000-000000000001');

-- Streaming (cat 1)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000001', 'YouTube',         'https://www.youtube.com/',                          'a0000000-0000-0000-0000-000000000001', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000002', 'Twitch',          'https://www.twitch.tv/',                            'a0000000-0000-0000-0000-000000000001', 2);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000003', 'Netflix',         'https://www.netflix.com/browse',                     'a0000000-0000-0000-0000-000000000001', 3);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000004', 'Prime Video',     'https://www.primevideo.com/',                        'a0000000-0000-0000-0000-000000000001', 4);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000005', 'HBO Max',         'https://play.hbomax.com/page/urn:hbo:page:home',     'a0000000-0000-0000-0000-000000000001', 5);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000006', 'Apple TV+',       'https://tv.apple.com/',                              'a0000000-0000-0000-0000-000000000001', 6);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000007', 'Disney+',         'https://www.disneyplus.com/es-es/',                   'a0000000-0000-0000-0000-000000000001', 7);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000008', 'Crunchyroll',     'https://www.crunchyroll.com/es-es/',                  'a0000000-0000-0000-0000-000000000001', 8);

-- Compras (cat 2)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000009', 'Amazon',          'https://www.amazon.es/',                             'a0000000-0000-0000-0000-000000000002', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000010', 'Chollometro',     'https://www.chollometro.com/',                       'a0000000-0000-0000-0000-000000000002', 2);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000011', 'AliExpress',      'https://es.aliexpress.com/',                         'a0000000-0000-0000-0000-000000000002', 3);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000012', 'Temu',            'https://www.temu.com/',                              'a0000000-0000-0000-0000-000000000002', 4);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000013', 'PC Componentes',  'https://www.pccomponentes.com/',                     'a0000000-0000-0000-0000-000000000002', 5);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000014', 'MediaMarkt',      'https://www.mediamarkt.es/',                         'a0000000-0000-0000-0000-000000000002', 6);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000015', 'Laced Records',   'https://www.lacedrecords.co/',                       'a0000000-0000-0000-0000-000000000002', 7);

-- Social (cat 3)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000016', 'Twitter / X',     'https://twitter.com/home',                           'a0000000-0000-0000-0000-000000000003', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000017', 'Reddit',          'https://www.reddit.com/',                            'a0000000-0000-0000-0000-000000000003', 2);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000018', 'Menéame',         'https://www.meneame.net/',                           'a0000000-0000-0000-0000-000000000003', 3);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000019', 'Discord',         'https://discordapp.com/',                            'a0000000-0000-0000-0000-000000000003', 4);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000020', 'Instagram',       'https://www.instagram.com/',                         'a0000000-0000-0000-0000-000000000003', 5);

-- Tiendas de Juegos (cat 4)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000021', 'Steam',           'https://store.steampowered.com/?l=spanish',          'a0000000-0000-0000-0000-000000000004', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000022', 'Epic Games',      'https://store.epicgames.com/es-ES/',                  'a0000000-0000-0000-0000-000000000004', 2);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000023', 'Humble Bundle',   'https://www.humblebundle.com/',                      'a0000000-0000-0000-0000-000000000004', 3);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000024', 'Instant Gaming',  'https://www.instant-gaming.com/es',                   'a0000000-0000-0000-0000-000000000004', 4);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000025', 'Eneba',           'https://www.eneba.com/',                             'a0000000-0000-0000-0000-000000000004', 5);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000026', 'Ubisoft Store',   'https://store.ubi.com/es/home',                       'a0000000-0000-0000-0000-000000000004', 6);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000027', 'EA Games',        'https://www.ea.com/es-es/games',                      'a0000000-0000-0000-0000-000000000004', 7);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000028', 'SkidrowReloaded', 'https://www.skidrowreloaded.com/',                   'a0000000-0000-0000-0000-000000000004', 8);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000029', 'ElAmigos',        'https://elamigos.site/',                             'a0000000-0000-0000-0000-000000000004', 9);

-- Gaming (cat 5)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000030', 'ElOtroLado',      'https://www.elotrolado.net/foro_pc-juegos_62',       'a0000000-0000-0000-0000-000000000005', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000031', 'GX.games',        'https://gx.games/',                                  'a0000000-0000-0000-0000-000000000005', 2);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000032', 'HowLongToBeat',   'https://howlongtobeat.com/',                         'a0000000-0000-0000-0000-000000000005', 3);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000033', 'PlayClassicGames','https://playclassic.games/',                          'a0000000-0000-0000-0000-000000000005', 4);

-- World of Warcraft (cat 6)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000034', 'Warcraft Logs',   'https://www.warcraftlogs.com/',                      'a0000000-0000-0000-0000-000000000006', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000035', 'Raidbots',        'https://www.raidbots.com/simbot',                    'a0000000-0000-0000-0000-000000000006', 2);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000036', 'QE Live',         'https://questionablyepic.com/live/',                  'a0000000-0000-0000-0000-000000000006', 3);

-- Viajes (cat 7)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000037', 'Airbnb',          'https://www.airbnb.es/',                             'a0000000-0000-0000-0000-000000000007', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000038', 'Booking',         'https://booking.com/',                               'a0000000-0000-0000-0000-000000000007', 2);

-- Inversiones (cat 8)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000039', 'XTB xStation 5',  'https://xstation5.xtb.com/',                         'a0000000-0000-0000-0000-000000000008', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000040', 'Investing.com',   'https://es.investing.com/',                          'a0000000-0000-0000-0000-000000000008', 2);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000041', 'Google Finance',  'https://www.google.com/finance/?hl=es',               'a0000000-0000-0000-0000-000000000008', 3);

-- Banca (cat 9)
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000042', 'ING Direct',      'https://ing.ingdirect.es/pfm/#login',                 'a0000000-0000-0000-0000-000000000009', 1);
INSERT OR IGNORE INTO links (id, name, url, category_id, display_order) VALUES ('b0000000-0000-0000-0000-000000000043', 'Trade Republic',  'https://traderepublic.com/es-es',                      'a0000000-0000-0000-0000-000000000009', 2);
