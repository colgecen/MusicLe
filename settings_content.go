package main

import "MusicLe/state"

// policiesContent returns the MusicLe usage policies in the current language.
func policiesContent() string {
	switch state.Current.Language {
	case state.LangTurkish:
		return `MusicLe Kullanım Politikaları

1. Kişisel kullanım
MusicLe, yalnızca kişisel müzik arşivinizi oluşturmak ve yönetmek için
tasarlanmış ücretsiz, açık kaynaklı bir araçtır. Ticari amaçlı kullanım
önerilmez.

2. Telif hakkı ve yasal sorumluluk
İndirdiğiniz içeriklerin telif haklarına ve bulunduğunuz ülkenin yasalarına
uymak tamamen sizin sorumluluğunuzdadır. MusicLe, içeriğin yasal durumunu
denetlemez ve herhangi bir yasal tavsiye vermez.

3. Dağıtım yasağı
İndirilen dosyaları başkalarına satmak, yeniden paylaşmak veya telif hakkı
sahibinin izni olmadan kamuya açmak yasaktır. Araç yalnızca kendi
kütüphaneniz içindir.

4. Hizmet sağlayıcıların şartları
Spotify ve YouTube gibi kaynakların kullanım koşullarına uymak zorundasınız.
Bu araç, ilgili platformların resmi API'lerine veya kamuya açık bağlantı
noktalarına bağımlıdır ve platformlar değişiklik yaptığında çalışma
davranışı değişebilir.

5. Gizlilik
Kimlik bilgileri (örn. tarayıcı çerezleri) yalnızca oturum boyunca yerel
olarak kullanılır; MusicLe hiçbir kişisel veriyi harici sunucuya göndermez.
Yapılandırma yalnızca makinenizdeki config.json dosyasında saklanır.

6. Sorumluluk reddi
Bu yazılım "olduğu gibi" sunulur, garanti verilmez. Yanlış kullanımdan
kaynaklanan sonuçlardan geliştirici sorumlu tutulamaz.`
	case state.LangSpanish:
		return `Políticas de uso de MusicLe

1. Uso personal
MusicLe es una herramienta gratuita de código abierto diseñada solo para
crear y gestionar tu archivo personal de música. No se recomienda el uso
comercial.

2. Derechos de autor y responsabilidad legal
Respetar los derechos de autor y las leyes de tu país es totalmente tu
responsabilidad. MusicLe no supervisa el estado legal del contenido ni
ofrece asesoramiento legal.

3. Prohibición de distribución
Está prohibido vender, recompartir o publicar los archivos descargados sin
permiso del titular. La herramienta es solo para tu biblioteca.

4. Condiciones de los proveedores
Debes cumplir las condiciones de fuentes como Spotify y YouTube. La
herramienta depende de sus APIs o endpoints públicos y puede cambiar si
ellos cambian.

5. Privacidad
Las credenciales (p. ej. cookies del navegador) solo se usan localmente
durante la sesión; MusicLe no envía datos personales a ningún servidor.
La configuración solo se guarda en config.json en tu máquina.

6. Exención de responsabilidad
Este software se ofrece "tal cual", sin garantía. El desarrollador no es
responsable de las consecuencias del mal uso.`
	case state.LangGerman:
		return `MusicLe Nutzungsrichtlinien

1. Persönliche Nutzung
MusicLe ist ein kostenloses Open-Source-Tool nur zum Erstellen und
Verwalten deines persönlichen Musikarchivs. Kommerzielle Nutzung wird
nicht empfohlen.

2. Urheberrecht und rechtliche Verantwortung
Die Einhaltung von Urheberrechten und Gesetzen deines Landes liegt voll
in deiner Verantwortung. MusicLe prüft den Rechtsstatus nicht und gibt
keine Rechtsberatung.

3. Verbreitungsverbot
Verkauf, Weiterverbreitung oder Veröffentlichung ohne Erlaubnis des
Rechteinhabers ist verboten. Das Tool ist nur für deine Bibliothek.

4. Bedingungen der Anbieter
Du musst die Bedingungen von Quellen wie Spotify und YouTube einhalten.
Das Tool hängt von deren APIs oder öffentlichen Endpunkten ab und kann
sich bei Änderungen dort anders verhalten.

5. Datenschutz
Anmeldedaten (z. B. Browser-Cookies) werden nur lokal während der Sitzung
verwendet; MusicLe sendet keine persönlichen Daten an Server. Die
Konfiguration liegt nur in config.json auf deinem Rechner.

6. Haftungsausschluss
Diese Software wird "wie sie ist" ohne Garantie bereitgestellt. Der
Entwickler haftet nicht für Folgen falscher Nutzung.`
	case state.LangFrench:
		return `Politiques d'utilisation de MusicLe

1. Usage personnel
MusicLe est un outil gratuit et open source conçu uniquement pour créer
et gérer votre bibliothèque musicale personnelle. L'usage commercial est
déconseillé.

2. Droits d'auteur et responsabilité légale
Respecter les droits d'auteur et les lois de votre pays relève entièrement
de votre responsabilité. MusicLe ne contrôle pas le statut légal du contenu
et ne donne aucun conseil juridique.

3. Interdiction de distribution
Il est interdit de vendre, repartager ou publier les fichiers téléchargés
sans l'autorisation du titulaire. L'outil est réservé à votre bibliothèque.

4. Conditions des fournisseurs
Vous devez respecter les conditions de sources comme Spotify et YouTube.
L'outil dépend de leurs APIs ou points publics et peut changer s'ils
changent.

5. Confidentialité
Les identifiants (ex. cookies du navigateur) ne sont utilisés que
localement pendant la session ; MusicLe n'envoie aucune donnée personnelle
vers un serveur. La configuration est stockée dans config.json sur votre
machine.

6. Clause de non-responsabilité
Ce logiciel est fourni "tel quel", sans garantie. Le développeur n'est pas
responsable des conséquences d'une mauvaise utilisation.`
	case state.LangArabic:
		return `سياسات استخدام MusicLe

1. الاستخدام الشخصي
MusicLe أداة مجانية مفتوحة المصدر مصممة فقط لإنشاء وإدارة مكتبتك
الموسيقية الشخصية. لا يُنصح بالاستخدام التجاري.

2. حقوق النشر والمسؤولية القانونية
الالتزام بحقوق النشر وقوانين بلدك مسؤوليتك الكاملة. لا يتحقق MusicLe من
الوضع القانوني للمحتوى ولا يقدم أي استشارة قانونية.

3. حظر التوزيع
يُحظر بيع الملفات المنزلة أو إعادة مشاركتها أو نشرها دون إذن صاحب الحق.
الأداة لمكتبتك الخاصة فقط.

4. شروط مزودي الخدمة
يجب الالتزام بشروط مصادر مثل Spotify وYouTube. تعتمد الأداة على واجهاتها
أو نقاطها العامة وقد يتغير سلوكها عند تغييرها.

5. الخصوصية
تُستخدم بيانات الاعتماد (مثل كوكيز المتصفح) محلياً فقط أثناء الجلسة؛ لا
يرسل MusicLe أي بيانات شخصية إلى خوادم خارجية. تُحفظ الإعدادات في
config.json على جهازك فقط.

6. إخلاء المسؤولية
يُقدم هذا البرنامج "كما هو" دون ضمان. المطور غير مسؤول عن نتائج سوء
الاستخدام.`
	case state.LangPortuguese:
		return `Políticas de uso do MusicLe

1. Uso pessoal
O MusicLe é uma ferramenta gratuita de código aberto feita apenas para
criar e gerir o teu arquivo pessoal de música. O uso comercial não é
recomendado.

2. Direitos de autor e responsabilidade legal
Respeitar os direitos de autor e as leis do teu país é totalmente tua
responsabilidade. O MusicLe não verifica o estado legal do conteúdo nem
dá aconselhamento jurídico.

3. Proibição de distribuição
É proibido vender, repartilhar ou publicar os ficheiros descarregados sem
permissão do titular. A ferramenta é só para a tua biblioteca.

4. Condições dos fornecedores
Tens de cumprir as condições de fontes como Spotify e YouTube. A ferramenta
depende das suas APIs ou endpoints públicos e pode mudar se eles mudarem.

5. Privacidade
As credenciais (ex. cookies do navegador) só são usadas localmente durante
a sessão; o MusicLe não envia dados pessoais para nenhum servidor. A
configuração fica só em config.json na tua máquina.

6. Isenção de responsabilidade
Este software é fornecido "como está", sem garantia. O programador não é
responsável pelas consequências do mau uso.`
	case state.LangChinese:
		return `MusicLe 使用政策

1. 个人使用
MusicLe 是仅用于创建和管理个人音乐档案的免费开源工具。不建议商业用途。

2. 版权与法律责任
遵守所下载内容的版权及所在国家法律完全是您的责任。MusicLe 不审查内容的
合法状态，也不提供任何法律建议。

3. 禁止分发
未经版权持有人许可，禁止出售、再分享或公开发布下载的文件。该工具仅用于
您自己的曲库。

4. 服务商条款
您必须遵守 Spotify、YouTube 等来源的使用条款。本工具依赖其官方 API 或
公开接口，平台变更时行为可能发生变化。

5. 隐私
凭证（如浏览器 Cookie）仅在会话期间本地使用；MusicLe 不会向外部服务器
发送任何个人数据。配置仅保存在本机的 config.json 文件中。

6. 免责声明
本软件按“现状”提供，不作任何保证。因误用导致的后果开发者不承担责任。`
	case state.LangJapanese:
		return `MusicLe 利用ポリシー

1. 個人利用
MusicLeは個人の音楽アーカイブの作成・管理専用の無料オープンソースツール
です。商用利用は推奨されません。

2. 著作権と法的責任
ダウンロードしたコンテンツの著作権およびお住まいの国の法律を守ることは
すべて利用者の責任です。MusicLeは合法性を確認せず、法的助言もしません。

3. 配布禁止
権利者の許可なくファイルを販売・再共有・公開することは禁止です。本ツール
は自分のライブラリ専用です。

4. サービス提供者の条件
SpotifyやYouTube等の利用条件を守る必要があります。本ツールは各プラット
フォームの公式APIや公開エンドポイントに依存し、仕様変更で動作が変わる
ことがあります。

5. プライバシー
認証情報（ブラウザCookie等）はセッション中のみローカルで使用し、MusicLe
は個人データを外部サーバーに送信しません。設定は端末内のconfig.jsonに
のみ保存されます。

6. 免責事項
本ソフトウェアは「現状のまま」提供され、保証はありません。誤用による結果
について開発者は責任を負いません。`
	case state.LangItalian:
		return `Politiche di utilizzo di MusicLe

1. Uso personale
MusicLe è uno strumento gratuito e open source pensato solo per creare e
gestire il tuo archivio musicale personale. L'uso commerciale è sconsigliato.

2. Diritto d'autore e responsabilità legale
Rispettare il diritto d'autore e le leggi del tuo paese è interamente tua
responsabilità. MusicLe non verifica lo stato legale dei contenuti e non
fornisce consulenza legale.

3. Divieto di distribuzione
È vietato vendere, ricondividere o pubblicare i file scaricati senza il
permesso del titolare. Lo strumento è solo per la tua libreria.

4. Condizioni dei fornitori
Devi rispettare le condizioni di fonti come Spotify e YouTube. Lo strumento
dipende dalle loro API o endpoint pubblici e può cambiare se cambiano loro.

5. Privacy
Le credenziali (es. cookie del browser) sono usate solo localmente durante
la sessione; MusicLe non invia dati personali ad alcun server. La
configurazione è salvata solo in config.json sulla tua macchina.

6. Esclusione di responsabilità
Questo software è fornito "così com'è", senza garanzia. Lo sviluppatore non
è responsabile delle conseguenze dell'uso improprio.`
	case state.LangRussian:
		return `Правила использования MusicLe

1. Личное использование
MusicLe — бесплатный инструмент с открытым кодом только для создания и
управления личным музыкальным архивом. Коммерческое использование не
рекомендуется.

2. Авторские права и юридическая ответственность
Соблюдение авторских прав и законов вашей страны — полностью ваша
ответственность. MusicLe не проверяет правовой статус контента и не даёт
юридических консультаций.

3. Запрет распространения
Запрещено продавать, распространять или публиковать скачанные файлы без
разрешения правообладателя. Инструмент только для вашей библиотеки.

4. Условия поставщиков
Вы обязаны соблюдать условия таких источников, как Spotify и YouTube.
Инструмент зависит от их API или публичных эндпоинтов и может измениться
при их изменениях.

5. Конфиденциальность
Учётные данные (напр. cookie браузера) используются только локально в
течение сессии; MusicLe не отправляет личные данные на внешние серверы.
Настройки хранятся только в config.json на вашем компьютере.

6. Отказ от ответственности
ПО предоставляется "как есть", без гарантий. Разработчик не отвечает за
последствия неправильного использования.`
	default:
		return `MusicLe Usage Policies

1. Personal use
MusicLe is a free, open-source tool designed only to create and manage
your personal music archive. Commercial use is not recommended.

2. Copyright and legal responsibility
Respecting copyrights and the laws of your country is entirely your
responsibility. MusicLe does not review the legal status of content and
gives no legal advice.

3. Distribution ban
Selling, re-sharing or publishing downloaded files without the rights
holder's permission is prohibited. The tool is only for your own library.

4. Provider terms
You must comply with the terms of sources like Spotify and YouTube. The
tool depends on their official APIs or public endpoints and behaviour may
change when they change.

5. Privacy
Credentials (e.g. browser cookies) are only used locally during the
session; MusicLe sends no personal data to any external server.
Configuration is stored only in config.json on your machine.

6. Disclaimer
This software is provided "as is", without warranty. The developer is not
liable for consequences of misuse.`
	}
}

// extrasContent returns the keyboard-shortcut reference and quick tips in the
// current language.
func extrasContent() string {
	switch state.Current.Language {
	case state.LangTurkish:
		return `MusicLe Ekstralar

Klavye Kısayolları
Genel:
- F1: Bölüm odağını değiştir (veya oynatma çubuğu odağını aç/kapat).
- F2: Görünüm arasında geçiş yap (Ana Sayfa, İndirme, Profil, Çalma Listesi,
  Ayarlar).
- F3: Ayarlar sekmesi arasında geçiş (yalnızca Ayarlar görünümünde).
- Esc: Ana Sayfa'ya dön.
- Ctrl+C: Çıkış (İndirme görünümündeyken indirmeyi iptal eder).

Oynatma Çubuğu (F1 ile odaklanın):
- ↑ / ↓: Ses seviyesini artır / azalt.
- ← / →: 5 saniye geri / ileri sar.
- Boşluk: Çal / Duraklat / Devam et.

İpuçları
- İndirilen parçalar otomatik olarak song_list.txt dosyasına kaydedilir ve
  Ana Sayfa'daki kitaplığınızda görünür.
- Ayarlar → Ses sekmesinden çıkış cihazını ve ses limitini seçebilirsiniz.
- Bir parça bittiğinde otomatik olarak bir sonrakine geçilir.
- Şarkı sözü ve spektrum görselleştirmesi oynatıcıda gösterilir.

Daha Fazla Bilgi
- Politikalar ve Hakkında sekmelerinde telif hakkı, gizlilik ve kullanılan
  3. taraf araçlar hakkında detaylı bilgi bulabilirsiniz.
- Tüm yapılandırma yalnızca makinenizdeki config.json dosyasında saklanır;
  hiçbir kişisel veri harici sunucuya gönderilmez.`
	case state.LangSpanish:
		return `MusicLe Extras

Atajos de teclado
General:
- F1: Cambiar el foco de sección (o activar/desactivar la barra).
- F2: Cambiar de vista (Inicio, Descarga, Perfil, Lista, Ajustes).
- F3: Cambiar de pestaña de ajustes (solo en Ajustes).
- Esc: Volver al Inicio.
- Ctrl+C: Salir (cancela la descarga en la vista de descarga).

Barra de reproducción (enfocar con F1):
- ↑ / ↓: Subir / bajar volumen.
- ← / →: Retroceder / avanzar 5 segundos.
- Espacio: Reproducir / Pausar / Continuar.

Consejos
- Las pistas descargadas se guardan en song_list.txt y aparecen en tu
  biblioteca del Inicio.
- En Ajustes → Sonido puedes elegir el dispositivo y el límite de volumen.
- Al terminar una pista se pasa automáticamente a la siguiente.
- Letras y espectro se muestran en el reproductor.

Más información
- En Políticas y Acerca de hay detalles de derechos, privacidad y
  herramientas de terceros.
- Todo se guarda solo en config.json en tu máquina.`
	case state.LangGerman:
		return `MusicLe Extras

Tastenkürzel
Allgemein:
- F1: Abschnittsfokus wechseln (oder Player-Leiste an/abschalten).
- F2: Ansicht wechseln (Start, Download, Profil, Playlist, Einstellungen).
- F3: Einstellungs-Tab wechseln (nur in Einstellungen).
- Esc: Zurück zur Startseite.
- Ctrl+C: Beenden (bricht Download in Download-Ansicht ab).

Player-Leiste (mit F1 fokussieren):
- ↑ / ↓: Lauter / leiser.
- ← / →: 5 Sekunden zurück / vor.
- Leertaste: Abspielen / Pause / Fortsetzen.

Tipps
- Downloads werden automatisch in song_list.txt gespeichert und erscheinen
  in deiner Bibliothek auf der Startseite.
- Unter Einstellungen → Ton kannst du Gerät und Lautstärkelimit wählen.
- Nach einem Titel wird automatisch zum nächsten gewechselt.
- Songtext und Spektrum werden im Player angezeigt.

Weitere Infos
- Unter Richtlinien und Über findest du Details zu Rechten, Datenschutz
  und Drittanbieter-Tools.
- Alles wird nur in config.json auf deinem Rechner gespeichert.`
	case state.LangFrench:
		return `MusicLe Extras

Raccourcis clavier
Général :
- F1 : Changer de section (ou activer/couper la barre de lecture).
- F2 : Changer de vue (Accueil, Téléchargement, Profil, Playlist, Réglages).
- F3 : Changer d'onglet de réglages (dans Réglages uniquement).
- Échap : Retour à l'Accueil.
- Ctrl+C : Quitter (annule le téléchargement dans la vue téléchargement).

Barre de lecture (focus avec F1) :
- ↑ / ↓ : Monter / baisser le volume.
- ← / → : Reculer / avancer de 5 secondes.
- Espace : Lecture / Pause / Reprendre.

Astuces
- Les morceaux téléchargés sont enregistrés dans song_list.txt et visibles
  dans votre bibliothèque d'Accueil.
- Dans Réglages → Son, choisissez la sortie et la limite de volume.
- À la fin d'un morceau, on passe automatiquement au suivant.
- Paroles et spectre affichés dans le lecteur.

Plus d'infos
- Onglets Politiques et À propos : droits, confidentialité et outils tiers.
- Tout est stocké uniquement dans config.json sur votre machine.`
	case state.LangArabic:
		return `إضافات MusicLe

اختصارات لوحة المفاتيح
عام:
- F1: تبديل التركيز (أو تفعيل/إيقاف شريط التشغيل).
- F2: التنقل بين الشاشات (الرئيسية، التنزيل، الملف، القائمة، الإعدادات).
- F3: التنقل بين تبويبات الإعدادات (في الإعدادات فقط).
- Esc: العودة للرئيسية.
- Ctrl+C: خروج (يلغي التنزيل في شاشة التنزيل).

شريط التشغيل (ركّز عليه بـ F1):
- ↑ / ↓: رفع / خفض الصوت.
- ← / →: ترجيع / تقديم 5 ثوانٍ.
- مسافة: تشغيل / إيقاف مؤقت / متابعة.

تلميحات
- تُحفظ المقاطع المنزلة تلقائياً في song_list.txt وتظهر في مكتبتك.
- من الإعدادات → الصوت اختر جهاز الإخراج وحد الصوت.
- عند انتهاء مقطع يتم الانتقال تلقائياً للتالي.
- تُعرض كلمات الأغاني والطيف في المشغل.

معلومات أكثر
- في تبويبي السياسات وحول تجد تفاصيل الحقوق والخصوصية وأدوات الطرف الثالث.
- كل الإعدادات في config.json على جهازك فقط.`
	case state.LangPortuguese:
		return `MusicLe Extras

Atalhos de teclado
Geral:
- F1: Mudar o foco (ou ligar/desligar a barra de reprodução).
- F2: Mudar de vista (Início, Download, Perfil, Playlist, Definições).
- F3: Mudar de aba das definições (só em Definições).
- Esc: Voltar ao Início.
- Ctrl+C: Sair (cancela o download na vista de download).

Barra de reprodução (focar com F1):
- ↑ / ↓: Aumentar / diminuir volume.
- ← / →: Recuar / avançar 5 segundos.
- Espaço: Tocar / Pausa / Continuar.

Dicas
- As faixas descarregadas ficam em song_list.txt e aparecem na tua
  biblioteca do Início.
- Em Definições → Som podes escolher o dispositivo e o limite de volume.
- Ao terminar uma faixa passa automaticamente à seguinte.
- Letras e espectro aparecem no leitor.

Mais informação
- Nos separadores Políticas e Sobre há detalhes de direitos, privacidade
  e ferramentas de terceiros.
- Tudo fica só em config.json na tua máquina.`
	case state.LangChinese:
		return `MusicLe 附加功能

键盘快捷键
通用：
- F1：切换焦点（或开/关播放条焦点）。
- F2：切换视图（首页、下载、个人资料、播放列表、设置）。
- F3：切换设置标签页（仅在设置中）。
- Esc：返回首页。
- Ctrl+C：退出（在下载视图中取消下载）。

播放条（用 F1 聚焦）：
- ↑ / ↓：增大 / 减小音量。
- ← / →：后退 / 前进 5 秒。
- 空格：播放 / 暂停 / 继续。

小贴士
- 下载的歌曲自动保存到 song_list.txt 并显示在首页曲库中。
- 在设置 → 声音中可选择输出设备和音量限制。
- 一曲结束会自动播放下一曲。
- 播放器中显示歌词与频谱可视化。

更多信息
- 在政策与关于页可查看版权、隐私及第三方工具详情。
- 所有配置仅保存在本机 config.json，不会外传。`
	case state.LangJapanese:
		return `MusicLe 追加機能

キーボードショートカット
一般:
- F1: セクション切替（再生バーのフォーカス切替）。
- F2: ビュー切替（ホーム、ダウンロード、プロフィール、プレイリスト、設定）。
- F3: 設定タブ切替（設定画面のみ）。
- Esc: ホームに戻る。
- Ctrl+C: 終了（ダウンロード画面では中断）。

再生バー（F1でフォーカス）:
- ↑ / ↓: 音量アップ / ダウン。
- ← / →: 5秒戻る / 進む。
- Space: 再生 / 一時停止 / 再開。

ヒント
- ダウンロード曲はsong_list.txtに自動保存されホームに表示されます。
- 設定 → サウンドで出力デバイスと音量制限を選べます。
- 曲終了後は自動で次曲へ進みます。
- 歌詞とスペクトラムがプレイヤーに表示されます。

詳細
- ポリシー・についてタブに権利・プライバシー・第三者ツールの詳細あり。
- 設定は端末内config.jsonのみに保存されます。`
	case state.LangItalian:
		return `MusicLe Extra

Scorciatoie tastiera
Generali:
- F1: Cambia sezione (o attiva/disattiva la barra di riproduzione).
- F2: Cambia vista (Home, Download, Profilo, Playlist, Impostazioni).
- F3: Cambia scheda impostazioni (solo in Impostazioni).
- Esc: Torna alla Home.
- Ctrl+C: Esci (annulla il download nella vista download).

Barra di riproduzione (focus con F1):
- ↑ / ↓: Alza / abbassa il volume.
- ← / →: Indietro / avanti di 5 secondi.
- Spazio: Riproduci / Pausa / Riprendi.

Suggerimenti
- I brani scaricati finiscono in song_list.txt e appaiono nella libreria Home.
- In Impostazioni → Audio scegli dispositivo e limite volume.
- Alla fine di un brano si passa al successivo.
- Testi e spettro mostrati nel player.

Altre info
- Nelle schede Politiche e Info trovi diritti, privacy e tool di terze parti.
- Tutto è salvato solo in config.json sulla tua macchina.`
	case state.LangRussian:
		return `MusicLe Дополнительно

Горячие клавиши
Общее:
- F1: Сменить фокус (или вкл/выкл фокус панели плеера).
- F2: Сменить вид (Главная, Загрузки, Профиль, Плейлист, Настройки).
- F3: Сменить вкладку настроек (только в Настройках).
- Esc: Назад на Главную.
- Ctrl+C: Выход (в виде загрузок отменяет загрузку).

Панель плеера (фокус через F1):
- ↑ / ↓: Громче / тише.
- ← / →: Назад / вперёд на 5 секунд.
- Пробел: Играть / Пауза / Продолжить.

Советы
- Скачанные треки сохраняются в song_list.txt и видны в библиотеке Главной.
- В Настройки → Звук выберите устройство и лимит громкости.
- После трека автоматически идёт следующий.
- Текст песни и спектр показываются в плеере.

Подробнее
- Во вкладках Правила и О приложении — права, приватность и сторонние tools.
- Всё хранится только в config.json на вашем компьютере.`
	default:
		return `MusicLe Extras

Keyboard Shortcuts
General:
- F1: Cycle section focus (or toggle player-bar focus).
- F2: Cycle views (Home, Downloads, Profile, Playlist, Settings).
- F3: Cycle settings tabs (Settings view only).
- Esc: Back to Home.
- Ctrl+C: Quit (cancels download in Downloads view).

Player Bar (focus with F1):
- Up / Down: Volume up / down.
- Left / Right: Seek back / forward 5 seconds.
- Space: Play / Pause / Resume.

Tips
- Downloaded tracks are saved to song_list.txt and appear in your Home library.
- In Settings → Sound you can pick the output device and volume limit.
- When a track ends it auto-advances to the next one.
- Lyrics and spectrum visualization show in the player.

More Info
- Policies and About tabs have copyright, privacy and 3rd-party tool details.
- All config is stored only in config.json on your machine.`
	}
}

// aboutContent returns the project information in the current language.
func aboutContent() string {
	switch state.Current.Language {
	case state.LangTurkish:
		return `MusicLe Hakkında

Proje ne zaman başladı?
26 Mart 2026'da ilk kod yapısı oluşturularak geliştirmeye başlandı.

İlk commit ne zaman atıldı?
26 Mart 2026 — "File Structure Changes" (Alperen Çölgeçen).

Kim tarafından kurulup yönetiliyor?
Kurucu ve geliştirici: Alperen ÇÖLGEÇEN
GitHub: @colgecen

Bu araç neye hizmet eder?
Kullanıcının kendi kişisel müzik kütüphanesini Spotify ve YouTube gibi
kaynaklardan indirip yerel olarak yönetmesine yarar.

Amaç?
Mevcut müzik akış platformlarındaki parçaları kendi kontrolünüzdeki bir
yerel arşive dönüştürmek; çevrimdışı dinleme, düzenleme ve kalıcı saklama
sağlamak. MusicLe bir akış servisi değildir, bir kişisel arşiv yöneticisidir.

Temel özellikler
- Spotify ve YouTube desteği ile parça ve çalma listesi indirme.
- Otomatik MP3 dönüştürme, ID3 etiketleme ve kapak resmi işleme.
- Profil ve çalma listesi yönetimi; şarkılar song_list.txt ile izlenir.
- Tema ve dil seçenekleri (11 dil desteği).
- Ses sekmesi: çıkış cihazı seçimi ve ses limiti ayarı.
- Spektrum/görselleştirme ve kayan şarkı sözü gibi oynatıcı özellikleri.

Kullanılan 3. taraf araçlar
- TerminalDownloader: saf Go indirme ve MP3 kodlama motoru.
- Spotify: meta veri ve arama için veri kaynağı.`
	case state.LangSpanish:
		return `Acerca de MusicLe

¿Cuándo empezó el proyecto?
El 26 de marzo de 2026 con la primera estructura de código.

¿Primer commit?
26 de marzo de 2026 — "File Structure Changes" (Alperen Çölgeçen).

¿Quién lo crea y mantiene?
Fundador y desarrollador: Alperen ÇÖLGEÇEN
GitHub: @colgecen

¿Para qué sirve?
Para descargar tu biblioteca personal desde Spotify y YouTube y
gestionarla en local.

Objetivo
Convertir piezas de streaming en un archivo local bajo tu control;
escucha sin conexión y almacenamiento permanente. No es un servicio de
streaming, es un gestor de archivo personal.

Características
- Descarga de pistas y listas desde Spotify y YouTube.
- Conversión MP3, etiquetas ID3 y carátulas automáticas.
- Perfiles y listas; seguimiento en song_list.txt.
- Temas e idiomas (11 idiomas).
- Sonido: dispositivo de salida y límite de volumen.
- Espectro y letras sincronizadas.

Terceros
- TerminalDownloader: motor de descarga y codificación MP3 en Go puro.
- Spotify: metadatos y búsqueda.`
	case state.LangGerman:
		return `Über MusicLe

Wann startete das Projekt?
Am 26. März 2026 mit der ersten Code-Struktur.

Erster Commit?
26. März 2026 — "File Structure Changes" (Alperen Çölgeçen).

Wer erstellt und pflegt es?
Gründer und Entwickler: Alperen ÇÖLGEÇEN
GitHub: @colgecen

Wozu dient es?
Zum Herunterladen deiner persönlichen Musikbibliothek von Spotify und
YouTube und zur lokalen Verwaltung.

Ziel
Streaming-Titel in ein lokales Archiv unter deiner Kontrolle verwandeln;
offline hören, bearbeiten und dauerhaft speichern. Kein Streaming-Dienst,
sondern ein Archiv-Manager.

Funktionen
- Track- und Playlist-Download von Spotify und YouTube.
- Automatische MP3-Konvertierung, ID3-Tags und Cover.
- Profile und Playlists; Tracking in song_list.txt.
- Themes und Sprachen (11 Sprachen).
- Ton: Ausgabegerät und Lautstärkelimit.
- Spektrum und scrollende Lyrics.

Drittanbieter
- TerminalDownloader: Pure-Go-Download- und MP3-Enkodierung.
- Spotify: Metadaten und Suche.`
	case state.LangFrench:
		return `À propos de MusicLe

Quand le projet a-t-il commencé ?
Le 26 mars 2026 avec la première structure de code.

Premier commit ?
26 mars 2026 — "File Structure Changes" (Alperen Çölgeçen).

Qui le crée et le maintient ?
Fondateur et développeur : Alperen ÇÖLGEÇEN
GitHub : @colgecen

À quoi sert-il ?
À télécharger votre bibliothèque personnelle depuis Spotify et YouTube
et à la gérer en local.

Objectif
Transformer des morceaux de streaming en archive locale sous votre
contrôle ; écoute hors ligne et stockage durable. Ce n'est pas un service
de streaming, c'est un gestionnaire d'archive.

Fonctionnalités
- Téléchargement de morceaux et playlists Spotify/YouTube.
- Conversion MP3 auto, tags ID3 et pochettes.
- Profils et playlists ; suivi dans song_list.txt.
- Thèmes et langues (11 langues).
- Son : sortie et limite de volume.
- Spectre et paroles défilantes.

Tiers
- TerminalDownloader : moteur de téléchargement et d’encodage MP3 en Go pur.
- Spotify : métadonnées et recherche.`
	case state.LangArabic:
		return `حول MusicLe

متى بدأ المشروع؟
في 26 مارس 2026 بأول هيكل برمجي.

أول commit؟
26 مارس 2026 — "File Structure Changes" (Alperen Çölgeçen).

من يؤسسه ويديره؟
المؤسس والمطور: Alperen ÇÖLGEÇEN
GitHub: @colgecen

ما فائدته؟
لتنزيل مكتبتك الموسيقية الشخصية من Spotify وYouTube وإدارتها محلياً.

الهدف
تحويل مقاطع البث إلى أرشيف محلي تحت سيطرتك؛ استماع دون اتصال وحفظ دائم.
ليس خدمة بث بل مدير أرشيف شخصي.

الميزات
- تنزيل المقاطع والقوائم من Spotify وYouTube.
- تحويل MP3 تلقائي ووسوم ID3 وأغلفة.
- ملفات وقوائم؛ التتبع في song_list.txt.
- سمات ولغات (11 لغة).
- الصوت: جهاز الإخراج وحد الصوت.
- الطيف وكلمات متحركة.

أدوات خارجية
- TerminalDownloader: محرك تنزيل وترميز MP3 بلغة Go الخالصة.
- Spotify: بيانات وصفية وبحث.`
	case state.LangPortuguese:
		return `Sobre o MusicLe

Quando começou o projeto?
A 26 de março de 2026 com a primeira estrutura de código.

Primeiro commit?
26 de março de 2026 — "File Structure Changes" (Alperen Çölgeçen).

Quem o cria e mantém?
Fundador e programador: Alperen ÇÖLGEÇEN
GitHub: @colgecen

Para que serve?
Para descarregar a tua biblioteca pessoal do Spotify e YouTube e geri-la
localmente.

Objetivo
Transformar faixas de streaming num arquivo local sob o teu controlo;
audição offline e armazenamento permanente. Não é streaming, é um gestor
de arquivo pessoal.

Funcionalidades
- Download de faixas e playlists do Spotify e YouTube.
- Conversão MP3 automática, tags ID3 e capas.
- Perfis e playlists; registo em song_list.txt.
- Temas e idiomas (11 idiomas).
- Som: dispositivo de saída e limite de volume.
- Espectro e letras rolantes.

Terceiros
- TerminalDownloader: motor de download e codificação MP3 em Go puro.
- Spotify: metadados e pesquisa.`
	case state.LangChinese:
		return `关于 MusicLe

项目何时开始？
2026年3月26日，首个代码结构搭建时开始。

首次提交？
2026年3月26日 — “File Structure Changes”（Alperen Çölgeçen）。

谁创建并维护？
创始人兼开发者：Alperen ÇÖLGEÇEN
GitHub：@colgecen

用途？
从 Spotify、YouTube 下载个人音乐库并在本地管理。

目标
将流媒体歌曲转为由你掌控的本地档案；离线收听、编辑与永久保存。
不是流媒体服务，而是个人档案管理器。

主要功能
- 支持 Spotify、YouTube 的单曲与歌单下载。
- 自动 MP3 转换、ID3 标签与封面处理。
- 个人资料与歌单管理；song_list.txt 跟踪。
- 主题与语言（11 种语言）。
- 声音：输出设备选择与音量限制。
- 频谱可视化与滚动歌词。

第三方
- TerminalDownloader：纯 Go 下载与 MP3 编码引擎。
- Spotify：元数据与搜索数据源。`
	case state.LangJapanese:
		return `MusicLe について

プロジェクト開始は？
2026年3月26日に最初のコード構造で開発開始。

初回コミットは？
2026年3月26日 — "File Structure Changes" (Alperen Çölgeçen)。

制作者・運営は？
創設者兼開発者: Alperen ÇÖLGEÇEN
GitHub: @colgecen

用途は？
SpotifyやYouTubeから個人の音楽ライブラリをダウンロードしローカル管理。

目的
配信曲を自分の管理下にあるローカルアーカイブ化し、オフライン再生・編集・
永久保存を実現。配信サービスではなく個人アーカイブマネージャー。

主な機能
- Spotify/YouTubeの曲・プレイリスト下载対応。
- MP3自動変換、ID3タグ、ジャケット処理。
- プロフィール・プレイリスト管理；song_list.txtで追跡。
- テーマと言語（11言語）。
- サウンド：出力デバイス選択と音量制限。
- スペクトラムとスクロール歌詞。

サードパーティ
- TerminalDownloader: ピュアGoのダウンロード・MP3エンコードエンジン。
- Spotify: メタデータと検索ソース。`
	case state.LangItalian:
		return `Info su MusicLe

Quando è iniziato il progetto?
Il 26 marzo 2026 con la prima struttura di codice.

Primo commit?
26 marzo 2026 — "File Structure Changes" (Alperen Çölgeçen).

Chi lo crea e mantiene?
Fondatore e sviluppatore: Alperen ÇÖLGEÇEN
GitHub: @colgecen

A cosa serve?
A scaricare la tua libreria personale da Spotify e YouTube e gestirla in
locale.

Obiettivo
Trasformare brani in streaming in un archivio locale sotto il tuo controllo;
ascolto offline e conservazione permanente. Non è streaming, è un gestore
d'archivio personale.

Funzionalità
- Download di brani e playlist da Spotify e YouTube.
- Conversione MP3 automatica, tag ID3 e copertine.
- Profili e playlist; tracciamento in song_list.txt.
- Temi e lingue (11 lingue).
- Audio: dispositivo di uscita e limite volume.
- Spettro e testi scorrevoli.

Terze parti
- TerminalDownloader: motore di download e codifica MP3 in Go puro.
- Spotify: metadati e ricerca.`
	case state.LangRussian:
		return `О MusicLe

Когда начался проект?
26 марта 2026 года с первой структуры кода.

Первый коммит?
26 марта 2026 — "File Structure Changes" (Alperen Çölgeçen).

Кто создаёт и поддерживает?
Основатель и разработчик: Alperen ÇÖLGEÇEN
GitHub: @colgecen

Зачем нужен?
Чтобы скачивать личную музыкальную библиотеку со Spotify и YouTube и
управлять ею локально.

Цель
Превратить стриминговые треки в локальный архив под вашим контролем;
офлайн-прослушивание и постоянное хранение. Это не стриминг-сервис, а
менеджер личного архива.

Возможности
- Загрузка треков и плейлистов со Spotify и YouTube.
- Автоконвертация MP3, ID3-теги и обложки.
- Профили и плейлисты; учёт в song_list.txt.
- Темы и языки (11 языков).
- Звук: устройство вывода и лимит громкости.
- Спектр и бегущая строка текста песни.

Сторонние
- TerminalDownloader: движок загрузки и кодирования MP3 на чистом Go.
- Spotify: метаданные и поиск.`
	default:
		return `About MusicLe

When did the project start?
On 26 March 2026 with the first code structure.

First commit?
26 March 2026 — "File Structure Changes" (Alperen Çölgeçen).

Who builds and maintains it?
Founder and developer: Alperen ÇÖLGEÇEN
GitHub: @colgecen

What is it for?
To download your personal music library from Spotify and YouTube and
manage it locally.

Goal
Turn streaming tracks into a local archive under your control; offline
listening and permanent storage. Not a streaming service, a personal
archive manager.

Key features
- Track and playlist download from Spotify and YouTube.
- Auto MP3 conversion, ID3 tagging and cover handling.
- Profiles and playlists; tracked in song_list.txt.
- Themes and languages (11 languages).
- Sound: output device and volume limit.
- Spectrum and scrolling lyrics.

3rd party
- TerminalDownloader: pure-Go download and MP3 encoding engine.
- Spotify: metadata and search source.`
	}
}
