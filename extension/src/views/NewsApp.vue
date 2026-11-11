<template>
  <main
    :class="themeClass"
    style="display: flex; height: 100%; width: 100%; overflow: hidden; background: var(--news-bg); color: var(--news-fg)"
  >
    <!-- Sidebar -->
    <aside class="news-sidebar">
      <!-- Add feed: solo el botón; URL y credenciales se piden al pulsar (#49) -->
      <div class="news-sidebar-top">
        <button class="news-cta" type="button" @click="openAddFeed">
          <Plus style="width: 16px; height: 16px" />
          {{ $gettext('Add feed') }}
        </button>
      </div>

      <!-- Sidebar nav -->
      <nav class="news-sidebar-nav">
        <div class="news-section">
          <span class="news-section-label">{{ $gettext('Subscriptions') }}</span>
          <span style="display: flex; align-items: center; gap: 4px">
            <oc-button
              color-role="secondary"
              appearance="raw"
              :aria-label="$gettext('Refresh')"
              :title="$gettext('Refresh')"
              :disabled="refreshing"
              @click="refreshNow"
            >
              <RefreshCw style="width: 15px; height: 15px" :style="refreshing ? 'animation: news-spin 1s linear infinite' : ''" />
            </oc-button>
            <oc-button
              color-role="secondary"
              appearance="raw"
              :aria-label="$gettext('New folder')"
              :title="$gettext('New folder')"
              @click="createFolder"
            >
              <FolderPlus style="width: 16px; height: 16px" />
            </oc-button>
          </span>
        </div>

        <template v-for="entry in navEntries" :key="entry.key">
          <div
            class="news-nav-entry"
            :style="{
              background: isActive(entry) ? 'var(--oc-role-secondary-container)' : 'transparent',
              paddingLeft: 8 + (entry.depth ?? 0) * 20 + 'px',
              fontWeight: entry.unread ? 600 : 400
            }"
            @mouseenter="hovered = entry.key"
            @mouseleave="hovered = ''"
            @click="select(entry)"
          >
            <img
              v-if="entry.kind === 'feed' && entry.favicon"
              :src="entry.favicon"
              alt=""
              width="16"
              height="16"
              loading="lazy"
              style="width: 16px; height: 16px; border-radius: 4px; object-fit: contain; flex-shrink: 0"
            />
            <component :is="entry.icon" v-else style="width: 16px; height: 16px; opacity: 0.7; flex-shrink: 0" />
            <span style="flex: 1; min-width: 0; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              {{ entry.label }}
            </span>
            <span v-if="entry.error" :title="entry.error" style="flex-shrink: 0; display: inline-flex">
              <AlertTriangle style="width: 14px; height: 14px; color: var(--news-warning)" />
            </span>
            <span v-if="entry.unread" style="font-size: 12px; opacity: 0.6; flex-shrink: 0">{{ entry.unread }}</span>
            <oc-button
              v-if="entry.kind === 'feed' || entry.kind === 'folder' || entry.kind === 'search'"
              color-role="secondary"
              appearance="raw"
              style="flex-shrink: 0"
              :aria-label="$gettext('Options')"
              :title="$gettext('Options')"
              @click.stop="openMenu = openMenu === entry.key ? '' : entry.key"
            >
              <MoreHorizontal style="width: 16px; height: 16px" />
            </oc-button>
          </div>

          <!-- Context menu -->
          <div
            v-if="openMenu === entry.key"
            style="margin: 2px 8px; border: 1px solid var(--news-border-medium); border-radius: 8px; padding: 4px; display: flex; flex-direction: column; gap: 2px; background: var(--news-bg); box-shadow: 0 4px 12px var(--news-shadow)"
          >
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Rename')" @click="renameFeed(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Move to folder…')" @click="moveFeed(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Credentials…')" @click="openCredentials(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Filter articles…')" @click="openFilter(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Rules…')" @click="openRules(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Scraper…')" @click="openScraper(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Retention…')" @click="openRetention(entry)" />
            <MenuBtn v-if="entry.kind === 'folder'" :label="$gettext('Rename')" @click="renameFolder(entry)" />
            <MenuBtn v-if="entry.kind === 'folder'" :label="$gettext('New subfolder…')" @click="createSubfolder(entry)" />
            <MenuBtn
              v-if="entry.kind === 'feed' || entry.kind === 'folder'"
              :label="$gettext('Mark all read')"
              @click="markEntryRead(entry)"
            />
            <MenuBtn
              v-if="entry.kind === 'feed' || entry.kind === 'folder' || entry.kind === 'search'"
              :label="$gettext('Delete')"
              danger
              @click="deleteEntry(entry)"
            />
          </div>
        </template>
      </nav>

      <!-- Pie: Ajustes (OPML vive dentro del diálogo de ajustes) + atajos -->
      <div class="news-sidebar-bottom">
        <button class="news-quiet-action" type="button" @click="openSettings">
          <Settings style="width: 16px; height: 16px" />
          {{ $gettext('News settings') }}
        </button>
        <button class="news-quiet-action" type="button" @click="showShortcuts = true">
          <Keyboard style="width: 16px; height: 16px" />
          {{ $gettext('Keyboard shortcuts') }}
        </button>
        <input
          ref="opmlInputEl"
          type="file"
          accept=".opml,.xml,text/xml,text/x-opml"
          style="display: none"
          @change="importOpml"
        />
      </div>
    </aside>

    <!-- Lista -->
    <NoteList
      ref="noteListRef"
      v-model:show-all="showAll"
      v-model:oldest-first="oldestFirst"
      v-model:search-query="searchQuery"
      :items="items"
      :feeds="feeds"
      :loading="loading"
      :error="error"
      :title="currentTitle"
      :unread-count="unreadCount"
      :search-active="searchActive"
      :selected-id="detail?.id ?? null"
      :more-available="moreAvailable"
      :width="listWidth"
      @select="openItem"
      @toggle-unread="toggleUnread"
      @toggle-star="toggleStar"
      @mark-all-read="markAllRead"
      @search="doSearch"
      @clear-search="clearSearch"
      @save-search="saveCurrentSearch"
      @load-more="loadMore"
    />

    <!-- Divisor redimensionable de la columna de titulares -->
    <div
      ref="resizerEl"
      style="width: 7px; flex-shrink: 0; cursor: col-resize; background: transparent"
      :style="{ background: resizing ? 'rgba(0,100,200,0.25)' : 'transparent' }"
      @mousedown="resizeStart"
    ></div>

    <!-- Detalle -->
    <ReaderPane
      v-if="detail"
      :item="detail"
      :feeds="feeds"
      :body="detailBody"
      :reader="readerStyle"
      :shared="!!sharedUrl"
      :full-loading="fullLoading"
      :full-failed="fullFailed"
      :full-body-active="!!fullBody"
      @close="detail = null"
      @toggle-full="toggleFull"
      @toggle-unread="toggleUnread"
      @toggle-star="toggleStar"
      @toggle-share="toggleShare"
      @open-original="openOriginal"
    />

    <!-- Estado vacío: ningún artículo seleccionado -->
    <section
      v-else
      style="display: flex; flex-direction: column; flex: 1; min-width: 0; align-items: center; justify-content: center; gap: 8px; padding: 24px; text-align: center"
    >
      <Newspaper style="width: 64px; height: 64px; opacity: 0.25" />
      <p style="margin: 8px 0 0; font-size: 18px; font-weight: 600">{{ $gettext('No article selected') }}</p>
      <p style="margin: 0; font-size: 14px; opacity: 0.6">{{ $gettext('Please select an article from the list.') }}</p>
    </section>

    <!-- Filtro de artículos por feed (News 28.4.0) -->
    <div
      v-if="filterOpen"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Filter articles')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="closeFilter"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 420px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">
          {{ $gettext('Filter articles') }}
        </h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
          {{ $gettext('Hide articles whose title, body or URL contains any keyword (comma-separated, case-insensitive).') }}
        </p>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Title keywords') }}</label>
        <input
          v-model="filterForm.titleKeywords"
          type="text"
          placeholder="sponsored,ads,offer"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 10px"
        />
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Body keywords') }}</label>
        <input
          v-model="filterForm.bodyKeywords"
          type="text"
          placeholder="tracking,cookie"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 10px"
        />
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('URL keywords') }}</label>
        <input
          v-model="filterForm.urlKeywords"
          type="text"
          placeholder="utm_,/tag/"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 16px"
        />
        <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
          <oc-button v-if="filterHasFilter" color-role="secondary" appearance="raw" style="font-size: 13px" @click="clearFilter">
            {{ $gettext('Clear filter') }}
          </oc-button>
          <span style="flex: 1"></span>
          <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="closeFilter">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="filterSaving" @click="saveFilter">
            {{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Retención por feed -->
    <div
      v-if="retentionOpen"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Article retention')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="retentionOpen = false"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 380px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Article retention') }}</h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
          {{ $gettext('Days to keep read, non-starred articles for this feed. 0 = use the server default.') }}
        </p>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Retention (days)') }}</label>
        <input
          v-model.number="retentionForm"
          type="number"
          min="0"
          max="3650"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 16px"
        />
        <div style="display: flex; gap: 8px; justify-content: flex-end">
          <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="retentionOpen = false">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="retentionSaving" @click="saveRetention">
            {{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Ajustes de usuario -->
    <SettingsDialog
      v-if="settingsOpen"
      v-model="settingsForm"
      :saving="settingsSaving"
      @save="saveSettings"
      @cancel="settingsOpen = false"
      @export-opml="exportOpml"
      @import-opml="opmlInputEl?.click()"
      @open-global-rules="openGlobalRules"
      @open-auto-read="openAutoRead"
    />

    <!-- Añadir feed: URL y credenciales solo visibles al abrirlo (#49) -->
    <div
      v-if="addFeedOpen"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Add feed')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="addFeedOpen = false"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 420px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 12px; font-size: 15px; font-weight: 600">{{ $gettext('Add feed') }}</h3>
        <label for="news-add-feed-url" style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Feed URL') }}</label>
        <input
          id="news-add-feed-url"
          ref="addFeedUrlEl"
          v-model="newFeedUrl"
          type="url"
          :placeholder="$gettext('https://site.example/feed')"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 10px"
          @keydown.enter="subscribeFeed"
        />
        <label style="display: flex; align-items: center; gap: 6px; font-size: 12px; cursor: pointer; opacity: 0.8">
          <input v-model="newFeedAuth" type="checkbox" />
          {{ $gettext('Requires authentication') }}
        </label>
        <div v-if="newFeedAuth" style="display: flex; flex-direction: column; gap: 6px; margin-top: 6px">
          <input
            v-model="newFeedUser"
            type="text"
            autocomplete="off"
            :placeholder="$gettext('Username')"
            :aria-label="$gettext('Username')"
            style="font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit"
          />
          <input
            v-model="newFeedPass"
            type="password"
            autocomplete="new-password"
            :placeholder="$gettext('Password')"
            :aria-label="$gettext('Password')"
            style="font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit"
            @keydown.enter="subscribeFeed"
          />
        </div>
        <p v-if="addFeedError" style="margin: 12px 0 0; font-size: 12px; color: var(--news-error)">{{ addFeedError }}</p>
        <div style="display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px">
          <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="addFeedOpen = false">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="!newFeedUrl.trim() || subscribing" @click="subscribeFeed">
            {{ $gettext('Subscribe') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Selector de feeds descubiertos -->
    <div
      v-if="discoverPickerOpen"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Select a feed')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="discoverPickerOpen = false"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 440px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Select a feed') }}</h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
          {{ $gettext('Multiple feeds were found on this site.') }}
        </p>
        <fieldset style="border: 0; margin: 0 0 16px; padding: 0; max-height: 260px; overflow-y: auto">
          <label
            v-for="(f, i) in discoverCandidates"
            :key="f.url"
            style="display: flex; align-items: center; gap: 8px; padding: 6px 4px; border-radius: 6px; cursor: pointer; font-size: 13px"
          >
            <input v-model="discoverSelected" type="radio" name="discover-feed" :value="i" />
            <span style="flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              {{ f.title || f.url }}
            </span>
            <span style="font-size: 11px; opacity: 0.5; text-transform: uppercase">{{ f.type }}</span>
          </label>
        </fieldset>
        <div style="display: flex; gap: 8px; justify-content: flex-end">
          <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="discoverPickerOpen = false">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="discoverSubscribing" @click="subscribeDiscovered">
            {{ $gettext('Subscribe') }}
          </oc-button>
        </div>
      </div>
    </div>
    <!-- Diálogo de texto genérico (nueva carpeta / renombrar feed / renombrar carpeta) -->
    <TextPromptDialog
      v-if="textPrompt"
      v-model="textPrompt.value"
      :title="textPrompt.title"
      :saving="textPromptSaving"
      @confirm="confirmTextPrompt"
      @cancel="textPrompt = null"
    />

    <!-- Mover feed a carpeta -->
    <MoveFeedDialog
      v-if="moveOpen"
      v-model="moveFolderSel"
      :folders="folders"
      :saving="moveSaving"
      @confirm="confirmMove"
      @cancel="moveOpen = false"
    />

    <!-- Credenciales del feed (Basic auth) -->
    <FeedCredentialsDialog
      v-if="credOpen"
      v-model:user="credUser"
      v-model:pass="credPass"
      :has-auth="credHasAuth"
      :saving="credSaving"
      @confirm="saveCredentials"
      @remove="removeCredentials"
      @cancel="credOpen = false"
    />

    <!-- Reglas de auto-marcado como leído -->
    <div
      v-if="autoReadOpen"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Auto-read rules')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="autoReadOpen = false"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 460px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Auto-read rules') }}</h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.65">
          {{ $gettext('Articles whose title matches a rule are marked as read automatically when they arrive.') }}
        </p>
        <div v-if="autoReadRules.length" style="margin-bottom: 12px; max-height: 200px; overflow-y: auto">
          <div
            v-for="rule in autoReadRules"
            :key="rule.id"
            style="display: flex; align-items: center; gap: 8px; padding: 5px 6px; border-radius: 6px; font-size: 13px"
          >
            <span style="flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              <span style="opacity: 0.6">{{ autoReadFeedLabel(rule.feedId) }}</span>
              <code style="font-size: 12px">{{ rule.titlePattern }}</code>
            </span>
            <button
              style="background: none; border: 0; cursor: pointer; padding: 2px; display: inline-flex"
              :aria-label="$gettext('Delete rule')"
              :title="$gettext('Delete rule')"
              @click="removeAutoRead(rule)"
            >
              <X style="width: 15px; height: 15px; color: var(--news-muted)" />
            </button>
          </div>
        </div>
        <div style="display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px">
          <label style="display: block; font-size: 12px; margin-bottom: -6px">{{ $gettext('New rule (regex on the title)') }}</label>
          <div style="display: flex; gap: 8px; align-items: center">
            <input
              v-model="autoReadPattern"
              type="text"
              spellcheck="false"
              :placeholder="'(?i)urgente'"
              style="flex: 1; min-width: 0; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit"
              @keydown.enter="addAutoReadRule"
            />
            <select v-model="autoReadFeedSel" style="font-size: 12px; padding: 5px 4px" :aria-label="$gettext('Feed')">
              <option :value="0">{{ $gettext('All feeds') }}</option>
              <option v-for="fo in feeds" :key="fo.id" :value="fo.id">{{ fo.title }}</option>
            </select>
            <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="autoReadSaving" @click="addAutoReadRule">
              {{ $gettext('Add') }}
            </oc-button>
          </div>
        </div>
        <div style="display: flex; gap: 8px; justify-content: flex-end">
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" @click="autoReadOpen = false">
            {{ $gettext('Close') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Selector CSS de extracción por feed (#39) -->
    <div
      v-if="scraperOpen"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Article scraper')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="scraperOpen = false"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 420px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Article scraper') }}</h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.65; line-height: 1.5">
          {{ $gettext('CSS selector of the article body on the site, used before the automatic extraction when the feed only has summaries. Leave empty to use the automatic extraction.') }}
        </p>
        <input
          v-model="scraperForm"
          type="text"
          spellcheck="false"
          placeholder="div#articleBody, article"
          style="width: 100%; box-sizing: border-box; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 16px"
        />
        <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
          <oc-button v-if="scraperForm" color-role="secondary" appearance="raw" style="font-size: 13px; color: var(--news-error)" @click="scraperForm = ''">
            {{ $gettext('Clear') }}
          </oc-button>
          <span style="flex: 1"></span>
          <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="scraperOpen = false">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="scraperSaving" @click="saveScraper">
            {{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Reglas block/keep (feed o globales) -->
    <div
      v-if="rulesOpen"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Filter rules')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="rulesOpen = false"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 520px; max-width: 94vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">
          {{ rulesTitle }}
        </h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.65; line-height: 1.5">
          {{ $gettext('One rule per line: Field=regex (RE2). Fields: EntryTitle, EntryURL, EntryAuthor, EntryContent, EntryDate. For EntryDate: future, before:YYYY-MM-DD, after:YYYY-MM-DD, between:A,B or max-age:7d.') }}
        </p>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Block rules (matching articles are hidden)') }}</label>
        <textarea
          v-model="rulesForm.block"
          rows="4"
          spellcheck="false"
          style="width: 100%; box-sizing: border-box; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 12px"
        ></textarea>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Keep rules (only matching articles are kept)') }}</label>
        <textarea
          v-model="rulesForm.keep"
          rows="4"
          spellcheck="false"
          style="width: 100%; box-sizing: border-box; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 16px"
        ></textarea>
        <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
          <oc-button v-if="rulesHasRules" color-role="secondary" appearance="raw" style="font-size: 13px; color: var(--news-error)" @click="clearRules">
            {{ $gettext('Clear rules') }}
          </oc-button>
          <span style="flex: 1"></span>
          <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="rulesOpen = false">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="rulesSaving" @click="saveRules">
            {{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Atajos de teclado -->
    <div
      v-if="showShortcuts"
      role="dialog"
      aria-modal="true"
      :aria-label="$gettext('Keyboard shortcuts')"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="showShortcuts = false"
    >
      <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 460px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
        <h3 style="margin: 0 0 12px; font-size: 15px; font-weight: 600">{{ $gettext('Keyboard shortcuts') }}</h3>
        <dl style="margin: 0; font-size: 13px; display: grid; grid-template-columns: 120px 1fr; gap: 6px 12px; max-height: 60vh; overflow-y: auto">
          <template v-for="s in shortcutList" :key="s.keys">
            <dt style="font-weight: 600"><code>{{ s.keys }}</code></dt>
            <dd style="margin: 0; opacity: 0.85">{{ $gettext(s.label) }}</dd>
          </template>
        </dl>
        <div style="display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px">
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" @click="showShortcuts = false">
            {{ $gettext('Close') }}
          </oc-button>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
// useRouter SIEMPRE desde web-pkg (inyecta el router del host); el de
// vue-router no tiene inyección en el contexto de extensión (issues #002/#004)
import { useRouter } from '@opencloud-eu/web-pkg'
import { useGettext } from 'vue3-gettext'
import {
  Newspaper,
  Star,
  FolderOpen,
  FolderPlus,
  Plus,
  X,
  Rss,
  MoreHorizontal,
  RefreshCw,
  AlertTriangle,
  Settings,
  Podcast,
  Keyboard,
  Bookmark
} from 'lucide-vue-next'
import SettingsDialog from '../components/SettingsDialog.vue'
import NoteList from '../components/NoteList.vue'
import ReaderPane, { type ReaderStyle } from '../components/ReaderPane.vue'
import TextPromptDialog from '../components/TextPromptDialog.vue'
import MoveFeedDialog from '../components/MoveFeedDialog.vue'
import FeedCredentialsDialog from '../components/FeedCredentialsDialog.vue'
import { useNewsApi, Item, Feed, Folder, Selection, FeedFilter, UserSettings, DiscoveredFeed, SavedSearch, Rules, AutoReadRule } from '../api'

const { $gettext } = useGettext()
const router = useRouter()
const route = computed(() => router.currentRoute.value)
const api = useNewsApi()

// botón de menú contextual (componente local mínimo)
const MenuBtn = defineComponent({
  props: { label: { type: String, required: true }, danger: { type: Boolean, default: false } },
  emits: ['click'],
  setup(props, { emit }) {
    return () =>
      h(
        'button',
        {
          class: 'news-menu-btn',
          style: {
            appearance: 'none',
            background: 'transparent',
            border: 'none',
            font: 'inherit',
            textAlign: 'left',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            width: '100%',
            padding: '6px 10px',
            borderRadius: '6px',
            fontSize: '13px',
            color: props.danger ? 'var(--news-error)' : 'inherit'
          },
          onClick: (e: MouseEvent) => {
            e.stopPropagation()
            emit('click')
          }
        },
        props.label
      )
  }
})

const folders = ref<Folder[]>([])
const feeds = ref<Feed[]>([])
const starredCount = ref(0)
const savedSearches = ref<SavedSearch[]>([])
const items = ref<Item[]>([])
const detail = ref<Item | null>(null)
const fullBody = ref('')
const fullLoading = ref(false)
const fullFailed = ref(false)
const sharedUrl = ref('')
const loading = ref(false)
const error = ref('')
const newFeedUrl = ref('')
const newFeedAuth = ref(false)
const newFeedUser = ref('')
const newFeedPass = ref('')
const subscribing = ref(false)
// #49: el formulario de suscripción vive en un diálogo; los campos no se ven hasta abrirlo.
const addFeedOpen = ref(false)
const addFeedError = ref('')
const addFeedUrlEl = ref<HTMLInputElement | null>(null)

function openAddFeed() {
  addFeedError.value = ''
  addFeedOpen.value = true
  nextTick(() => addFeedUrlEl.value?.focus())
}
const refreshing = ref(false)
const openMenu = ref('')
const hovered = ref('')
const showAll = ref(false)
const oldestFirst = ref(false)
const moreAvailable = ref(false)
const opmlInputEl = ref<HTMLInputElement | null>(null)
const searchQuery = ref('')
const noteListRef = ref<InstanceType<typeof NoteList> | null>(null)

// Atajos de teclado (#34): índice del item "en foco" de la lista y ayuda.
const listIndex = ref(-1)
const showShortcuts = ref(false)

const shortcutList = [
  { keys: 'j / n / ↓', label: 'Next article' },
  { keys: 'k / p / ↑', label: 'Previous article' },
  { keys: 'o / Enter', label: 'Open article' },
  { keys: 'm', label: 'Toggle read/unread' },
  { keys: 'f', label: 'Star / unstar' },
  { keys: 'v', label: 'Open original link' },
  { keys: 'g u', label: 'All articles' },
  { keys: 'g b', label: 'Starred' },
  { keys: 'g p', label: 'Podcasts' },
  { keys: 'a', label: 'Mark all read' },
  { keys: 'r', label: 'Refresh feeds' },
  { keys: '/', label: 'Focus search' },
  { keys: '?', label: 'Show this help' },
  { keys: 'Esc', label: 'Close dialogs' }
]

// resize de la columna de titulares arrastrando el divisor derecho. El ancho
// seleccionado persiste en localStorage.
const listWidthKey = 'ocnews:listWidth'
const resizerEl = ref<HTMLElement | null>(null)
const listWidth = ref<number>(loadListWidth())
const resizing = ref(false)
const resizeStartX = ref(0)
const resizeStartW = ref(380)

function loadListWidth(): number {
  try {
    const v = parseInt(localStorage.getItem(listWidthKey) ?? '', 10)
    if (Number.isFinite(v) && v >= 240 && v <= 700) return v
  } catch {
    /* sin storage: usar default */
  }
  return 380
}

function saveListWidthLocal() {
  try {
    localStorage.setItem(listWidthKey, String(listWidth.value))
  } catch {
    /* sin storage */
  }
}

// persiste el ancho en el backend (user_settings) para que sobreviva a
// cualquier refresco, incluso si localStorage no está disponible en el iframe.
async function persistListWidth() {
  try {
    await api.updateSettings({ readerListWidth: String(listWidth.value) })
  } catch {
    /* best-effort: el local ya quedó guardado */
  }
}

function resizeStart(e: MouseEvent) {
  if (e.button !== 0) return
  resizing.value = true
  resizeStartX.value = e.clientX
  resizeStartW.value = listWidth.value
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', resizeMove)
  window.addEventListener('mouseup', resizeEnd)
}

function resizeMove(e: MouseEvent) {
  if (!resizing.value) return
  const dx = e.clientX - resizeStartX.value
  listWidth.value = Math.max(240, Math.min(700, resizeStartW.value + dx))
}

function resizeEnd() {
  resizing.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  window.removeEventListener('mousemove', resizeMove)
  window.removeEventListener('mouseup', resizeEnd)
  saveListWidthLocal()
  persistListWidth()
}

const searchActive = computed(() => searchQuery.value.trim() !== '')

const userTheme = ref('system')
const userWidth = ref('wide')
const userFont = ref('default')
const userFontSize = ref('15')

// Tema real: elige setting del usuario o prefiere sistema. Escucha cambios de
// prefers-color-scheme para que "System" funcione sin recargar.
const mediaDark = window.matchMedia('(prefers-color-scheme: dark)')
const systemDark = ref(mediaDark.matches)
function onMediaChange(e: MediaQueryListEvent | MediaQueryList) {
  systemDark.value = 'matches' in e ? e.matches : (e as MediaQueryList).matches
}
const effectiveTheme = computed(() => {
  if (userTheme.value === 'system') return systemDark.value ? 'dark' : 'light'
  return userTheme.value
})
const themeClass = computed(() => `news-theme-${effectiveTheme.value}`)

const readerClass = computed(() => `news-theme-${effectiveTheme.value}`)
const readerMaxWidth = computed(() => ({ narrow: '52ch', normal: '72ch', wide: '100%' })[userWidth.value] ?? '100%')
const readerFontFamily = computed(
  () =>
    ({
      default: '',
      serif: "Georgia, 'Times New Roman', serif",
      sans: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif",
      mono: "ui-monospace, 'Cascadia Mono', Menlo, Consolas, monospace"
    })[userFont.value] ?? ''
)
const readerFontSizePx = computed(() => `${userFontSize.value || '15'}px`)
const readerBg = computed(() =>
  effectiveTheme.value === 'dark' ? 'var(--news-bg)' : effectiveTheme.value === 'light' ? 'var(--news-bg)' : ''
)
const readerFg = computed(() =>
  effectiveTheme.value === 'dark' ? 'var(--news-fg)' : effectiveTheme.value === 'light' ? 'var(--news-fg)' : ''
)

const readerStyle = computed<ReaderStyle>(() => ({
  class: readerClass.value,
  maxWidth: readerMaxWidth.value,
  fontFamily: readerFontFamily.value,
  fontSizePx: readerFontSizePx.value,
  bg: readerBg.value,
  fg: readerFg.value
}))

async function loadSettings() {
  try {
    const s = await api.mySettings()
    userTheme.value = s.theme || 'system'
    userWidth.value = s.readerMaxWidth || 'wide'
    userFont.value = s.readerFont || 'default'
    userFontSize.value = s.readerFontSize || '15'
    // ancho de lista persistido en el backend (tras cualquier refresco)
    const w = parseInt(s.readerListWidth ?? '', 10)
    if (Number.isFinite(w) && w >= 240 && w <= 700) {
      listWidth.value = w
      saveListWidthLocal()
    }
  } catch {
    /* defaults */
  }
}

function doSearch() {
  if (searchQuery.value.trim()) {
    detail.value = null
    loadItems()
  }
}

function clearSearch() {
  searchQuery.value = ''
  detail.value = null
  loadItems()
}
const filterOpen = ref(false)
const filterSaving = ref(false)
const filterFeedId = ref<number | null>(null)
const filterForm = ref<Omit<FeedFilter, 'feedId'>>({ titleKeywords: '', bodyKeywords: '', urlKeywords: '' })

const filterHasFilter = computed(
  () =>
    filterForm.value.titleKeywords.trim() !== '' ||
    filterForm.value.bodyKeywords.trim() !== '' ||
    filterForm.value.urlKeywords.trim() !== ''
)

const retentionOpen = ref(false)
const retentionSaving = ref(false)
const retentionFeedId = ref<number | null>(null)
const retentionForm = ref(0)

const settingsOpen = ref(false)
const settingsSaving = ref(false)
const settingsForm = ref<UserSettings>({ theme: 'system', readerMaxWidth: 'wide', feedIntervalMin: '', readerFont: 'default', readerFontSize: '15', ntfyTopic: '' })

const discoverPickerOpen = ref(false)
const discoverCandidates = ref<DiscoveredFeed[]>([])
const discoverSelected = ref(0)
const discoverSubscribing = ref(false)

const BATCH = 50

type NavEntry = {
  key: string
  kind: 'all' | 'starred' | 'podcasts' | 'search' | 'folder' | 'feed'
  label: string
  icon: typeof Rss
  sel: Selection
  unread: number
  error?: string
  depth?: number
  favicon?: string
  searchQuery?: string
  searchId?: number
}

const selection = computed<Selection>(() => {
  const r = route.value
  if (r.name === 'news-feed') return { kind: 'feed', id: Number(r.params.feedId) }
  if (r.name === 'news-folder') return { kind: 'folder', id: Number(r.params.folderId) }
  if (r.name === 'news-starred') return { kind: 'starred' }
  if (r.name === 'news-podcasts') return { kind: 'podcasts' }
  return { kind: 'all' }
})

const podcastFeeds = computed(() => feeds.value.filter((f) => f.isPodcast))
const podcastIds = computed(() => new Set(podcastFeeds.value.map((f) => f.id)))

const navEntries = computed<NavEntry[]>(() => {
  const byFolder = new Map<number, Feed[]>()
  for (const f of feeds.value) {
    const k = f.folderId ?? 0
    byFolder.set(k, [...(byFolder.get(k) ?? []), f])
  }
  const entries: NavEntry[] = [
    { key: 'all', kind: 'all', label: $gettext('All articles'), icon: Newspaper, sel: { kind: 'all' }, unread: totalUnread.value },
    { key: 'starred', kind: 'starred', label: $gettext('Starred'), icon: Star, sel: { kind: 'starred' }, unread: starredCount.value },
    { key: 'podcasts', kind: 'podcasts', label: $gettext('Podcasts'), icon: Podcast, sel: { kind: 'podcasts' }, unread: 0 }
  ]
  for (const ss of savedSearches.value) {
    entries.push({
      key: `search-${ss.id}`,
      kind: 'search',
      label: ss.name,
      icon: Bookmark,
      sel: { kind: 'all' },
      unread: 0,
      searchQuery: ss.query,
      searchId: ss.id
    })
  }
  for (const folder of folders.value) {
    entries.push({
      key: `folder-${folder.id}`,
      kind: 'folder',
      label: folder.name,
      icon: FolderOpen,
      sel: { kind: 'folder', id: folder.id },
      unread: (byFolder.get(folder.id) ?? []).reduce((a, f) => a + f.unreadCount, 0)
    })
    for (const f of byFolder.get(folder.id) ?? []) {
      entries.push({
        key: `feed-${f.id}`,
        kind: 'feed',
        label: f.title,
        icon: Rss,
        sel: { kind: 'feed', id: f.id },
        unread: f.unreadCount,
        error: f.updateErrorCount > 0 ? f.lastUpdateError ?? `${f.updateErrorCount} errors` : undefined,
        depth: 1,
        favicon: f.faviconLink
      })
    }
  }
  // subcarpetas (#41): un nivel; su unread incluye sus feeds; el padre no
  // duplica contadores (los feeds viven en la subcarpeta).
  const rootFolders = folders.value.filter((fo) => !fo.parentId)
  const childByParent = new Map<number, Folder[]>()
  for (const fo of folders.value) {
    if (!fo.parentId) continue
    const k = fo.parentId
    childByParent.set(k, [...(childByParent.get(k) ?? []), fo])
  }
  for (const folder of rootFolders) {
    const children = childByParent.get(folder.id) ?? []
    const childUnread = children.reduce(
      (a, c) => a + (byFolder.get(c.id) ?? []).reduce((x, f) => x + f.unreadCount, 0),
      0
    )
    const parentEntry = entries.find((e) => e.key === `folder-${folder.id}`)
    if (parentEntry) parentEntry.unread += childUnread
    for (const child of children) {
      entries.push({
        key: `folder-${child.id}`,
        kind: 'folder',
        label: child.name,
        icon: FolderOpen,
        sel: { kind: 'folder', id: child.id },
        unread: (byFolder.get(child.id) ?? []).reduce((a, f) => a + f.unreadCount, 0),
        depth: 1
      })
      for (const f of byFolder.get(child.id) ?? []) {
        entries.push({
          key: `feed-${f.id}`,
          kind: 'feed',
          label: f.title,
          icon: Rss,
          sel: { kind: 'feed', id: f.id },
          unread: f.unreadCount,
          error: f.updateErrorCount > 0 ? f.lastUpdateError ?? `${f.updateErrorCount} errors` : undefined,
          depth: 2,
          favicon: f.faviconLink
        })
      }
    }
  }
  for (const f of byFolder.get(0) ?? []) {
    entries.push({
      key: `feed-${f.id}`,
      kind: 'feed',
      label: f.title,
      icon: Rss,
      sel: { kind: 'feed', id: f.id },
      unread: f.unreadCount,
      error: f.updateErrorCount > 0 ? f.lastUpdateError ?? `${f.updateErrorCount} errors` : undefined,
      depth: 1,
      favicon: f.faviconLink
    })
  }
  return entries
})

async function toggleFull() {
  if (!detail.value) return
  if (fullBody.value) {
    fullBody.value = '' // volver al resumen del feed
    return
  }
  fullLoading.value = true
  fullFailed.value = false
  try {
    const res = await api.itemFull(detail.value.id)
    fullBody.value = res.body
  } catch {
    fullFailed.value = true
  } finally {
    fullLoading.value = false
  }
}

// cuerpo del detalle: enlaces del artículo a pestaña nueva (dentro del host
// navegarían en el propio frame); las imágenes ya llegan vía proxy firmado
const detailBody = computed(() => {
  const base = fullBody.value || detail.value?.body || ""
  return base.replaceAll('<a ', '<a target="_blank" rel="noopener noreferrer" ')
})

const totalUnread = computed(() => feeds.value.reduce((a, f) => a + f.unreadCount, 0))
const unreadCount = computed(() =>
  selection.value.kind === 'starred' ? starredCount.value : items.value.filter((i) => i.unread).length
)
const currentTitle = computed(() => {
  const e = navEntries.value.find((e) => JSON.stringify(e.sel) === JSON.stringify(selection.value))
  return e?.label ?? $gettext('News')
})

function isActive(entry: NavEntry) {
  if (entry.kind === 'search') {
    return searchActive.value && searchQuery.value.trim() === entry.searchQuery
  }
  return JSON.stringify(entry.sel) === JSON.stringify(selection.value)
}

function select(entry: NavEntry) {
  detail.value = null
  if (entry.kind === 'search') {
    searchQuery.value = entry.searchQuery ?? ''
    loadItems()
    return
  }
  switch (entry.sel.kind) {
    case 'all': router.push('/news/'); break
    case 'starred': router.push('/news/starred'); break
    case 'podcasts': router.push('/news/podcasts'); break
    case 'feed': router.push(`/news/feed/${entry.sel.id}`); break
    case 'folder': router.push(`/news/folder/${entry.sel.id}`); break
  }
}

async function loadSidebar() {
  const [f, fd, ss] = await Promise.all([api.feeds(), api.folders(), api.searches()])
  feeds.value = f.feeds
  starredCount.value = f.starredCount
  folders.value = fd.folders
  savedSearches.value = ss.searches
}

async function loadItems(append = false) {
  loading.value = !append
  error.value = ''
  try {
    const getRead = selection.value.kind === 'starred' ? true : showAll.value
    if (searchActive.value) {
      const res = await api.search(searchQuery.value.trim(), { getRead, batchSize: 100, oldestFirst: oldestFirst.value })
      items.value = res.items
      moreAvailable.value = false
      if (!append) listIndex.value = -1
      return
    }
    const selForFetch: Selection = selection.value.kind === 'podcasts' ? { kind: 'all' } : selection.value
    const offset = append ? (oldestFirst.value ? Math.max(...items.value.map((i) => i.id)) : Math.min(...items.value.map((i) => i.id))) : undefined
    const res = await api.items(selForFetch, { getRead, batchSize: BATCH, oldestFirst: oldestFirst.value, offset })
    let fetched = append ? [...items.value, ...res.items] : res.items
    if (selection.value.kind === 'podcasts') {
      fetched = fetched.filter((i) => podcastIds.value.has(i.feedId))
    }
    items.value = fetched
    if (!append) listIndex.value = -1
    moreAvailable.value = res.items.length >= BATCH
  } catch (e) {
    error.value = $gettext('Could not load items: ') + errText(e)
  } finally {
    loading.value = false
  }
}

async function refreshCounts() {
  try {
    const f = await api.feeds()
    feeds.value = f.feeds
    starredCount.value = f.starredCount
  } catch {
    /* el sidebar se refrescará en el siguiente load */
  }
}

function navList(delta: number) {
  if (items.value.length === 0) return
  if (listIndex.value < 0) {
    listIndex.value = delta > 0 ? 0 : items.value.length - 1
  } else {
    listIndex.value = Math.max(0, Math.min(items.value.length - 1, listIndex.value + delta))
  }
  openItem(items.value[listIndex.value])
}

function focusSearch() {
  noteListRef.value?.focusSearch()
}

// handler global de teclado (#34). No interfiere con inputs (salvo Esc para
// cerrar diálogos) ni con la secuencia g+letra.
let gKeyAt = 0
function onKeydown(e: KeyboardEvent) {
  const t = e.target as HTMLElement | null
  const tag = t?.tagName ?? ''
  const isTyping =
    !!t && (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || t.isContentEditable)

  if (e.key === 'Escape') {
    if (showShortcuts.value) showShortcuts.value = false
    else if (filterOpen.value) filterOpen.value = false
    else if (retentionOpen.value) retentionOpen.value = false
    else if (settingsOpen.value) settingsOpen.value = false
    else if (discoverPickerOpen.value) discoverPickerOpen.value = false
    else if (addFeedOpen.value) addFeedOpen.value = false
    else if (textPrompt.value) textPrompt.value = null
    else if (moveOpen.value) moveOpen.value = false
    else if (credOpen.value) credOpen.value = false
    else if (openMenu.value) openMenu.value = ''
    return
  }
  if (isTyping) return
  if (e.ctrlKey || e.metaKey || e.altKey) return

  const key = e.key.toLowerCase()

  if (gKeyAt && Date.now() - gKeyAt < 1500) {
    gKeyAt = 0
    const target =
      key === 'u' ? navEntries.value.find((x) => x.kind === 'all') :
      key === 'b' ? navEntries.value.find((x) => x.kind === 'starred') :
      key === 'p' ? navEntries.value.find((x) => x.kind === 'podcasts') : undefined
    if (target) {
      e.preventDefault()
      select(target)
      return
    }
  }
  if (key === 'g') {
    gKeyAt = Date.now()
    return
  }

  switch (key) {
    case 'j':
    case 'n':
    case 'arrowdown':
      e.preventDefault()
      navList(1)
      break
    case 'k':
    case 'p':
    case 'arrowup':
      e.preventDefault()
      navList(-1)
      break
    case 'o':
      if (detail.value == null && items.value.length > 0) {
        e.preventDefault()
        navList(1)
      }
      break
    case 'enter':
      // solo cuando el foco no está en un botón/enlace (activarían su click)
      if (detail.value == null && items.value.length > 0 && tag !== 'BUTTON' && tag !== 'A') {
        e.preventDefault()
        navList(1)
      }
      break
    case 'm':
      if (detail.value) toggleUnread(detail.value)
      break
    case 'f':
      if (detail.value) toggleStar(detail.value)
      break
    case 'v':
      if (detail.value) {
        e.preventDefault()
        openOriginal()
      }
      break
    case 'a':
      if (unreadCount.value > 0) markAllRead()
      break
    case 'r':
      refreshNow()
      break
    case '/':
      e.preventDefault()
      focusSearch()
      break
    case '?':
      showShortcuts.value = !showShortcuts.value
      break
  }
}

async function openItem(item: Item) {
  listIndex.value = items.value.findIndex((x) => x.id === item.id)
  detail.value = item
  fullBody.value = ''
  fullFailed.value = false
  sharedUrl.value = ''
  if (item.unread) {
    item.unread = false
    await api.markRead(item.id)
    refreshCounts()
  }
}

async function toggleUnread(item: Item) {
  item.unread = !item.unread
  if (item.unread) {
    await api.markUnread(item.id)
  } else {
    await api.markRead(item.id)
  }
  refreshCounts()
}

async function toggleStar(item: Item) {
  item.starred = !item.starred
  if (item.starred) {
    await api.star(item.id)
  } else {
    await api.unstar(item.id)
  }
  refreshCounts()
}

async function refreshNow() {
  refreshing.value = true
  error.value = ''
  try {
    await api.refresh()
    await loadSidebar()
    await loadItems()
  } catch (e) {
    error.value = $gettext('Refresh failed: ') + errText(e)
  } finally {
    refreshing.value = false
  }
}

async function loadMore() {
  await loadItems(true)
}

async function markAllRead() {
  const newest = items.value.reduce((a, i) => Math.max(a, i.id), 0)
  await api.markAllRead(newest)
  await loadSidebar()
  await loadItems()
}

async function subscribeFeed() {
  const url = newFeedUrl.value.trim()
  if (!url) return
  subscribing.value = true
  addFeedError.value = ''
  const authUser = newFeedAuth.value ? newFeedUser.value.trim() : ''
  const authPass = newFeedAuth.value ? newFeedPass.value : ''
  try {
    await api.addFeed(url, null, authUser, authPass)
    newFeedUrl.value = ''
    newFeedAuth.value = false
    newFeedUser.value = ''
    newFeedPass.value = ''
    addFeedOpen.value = false
    await loadSidebar()
    await loadItems()
  } catch (e: unknown) {
    const resp = (e as { response?: { status?: number; data?: { error?: { code?: string; message?: string } } } })?.response
    if (resp?.data?.error?.code === 'feed_auth_required') {
      // el origen pide auth: abrir los campos de credenciales y avisar
      newFeedAuth.value = true
      addFeedError.value = resp.data.error.message || $gettext('This feed requires authentication (username and password)')
    } else if (resp?.status === 422) {
      // no es un feed directo: probar autodetección en la URL del sitio
      try {
        const res = await api.discover(url)
        if (res.feeds?.length === 1) {
          await api.addFeed(res.feeds[0].url, null, authUser, authPass)
          newFeedUrl.value = ''
          newFeedAuth.value = false
          newFeedUser.value = ''
          newFeedPass.value = ''
          addFeedOpen.value = false
          await loadSidebar()
          await loadItems()
          return
        }
        if (res.feeds?.length > 1) {
          discoverCandidates.value = res.feeds
          discoverSelected.value = 0
          discoverPickerOpen.value = true
          return
        }
      } catch {
        /* si discover también falla, caemos al error genérico */
      }
      addFeedError.value = extractErrorMessage(e, $gettext('Could not subscribe to the feed'))
    } else {
      addFeedError.value = extractErrorMessage(e, $gettext('Could not subscribe to the feed'))
    }
  } finally {
    subscribing.value = false
  }
}

// subscribeDiscovered: suscribe el feed elegido en el selector de descubrimiento.
async function subscribeDiscovered() {
  const feed = discoverCandidates.value[discoverSelected.value]
  if (!feed) {
    discoverPickerOpen.value = false
    return
  }
  discoverSubscribing.value = true
  try {
    await api.addFeed(feed.url, null, newFeedAuth.value ? newFeedUser.value.trim() : '', newFeedAuth.value ? newFeedPass.value : '')
    newFeedUrl.value = ''
    newFeedAuth.value = false
    newFeedUser.value = ''
    newFeedPass.value = ''
    discoverPickerOpen.value = false
    addFeedOpen.value = false
    await loadSidebar()
    await loadItems()
  } catch (e) {
    // cerrar el selector para que el error se vea en el diálogo de añadir
    discoverPickerOpen.value = false
    addFeedError.value = extractErrorMessage(e, $gettext('Could not subscribe to the feed'))
  } finally {
    discoverSubscribing.value = false
  }
}

// Diálogo de texto genérico (sustituye window.prompt, que el host puede
// suprimir): nueva carpeta, renombrar feed, renombrar carpeta, guardar búsqueda.
type TextPromptAction = 'newFolder' | 'newSubfolder' | 'renameFeed' | 'renameFolder' | 'saveSearch'
const textPrompt = ref<{ action: TextPromptAction; title: string; value: string; entryKey?: string } | null>(null)
const textPromptSaving = ref(false)

function saveCurrentSearch() {
  textPrompt.value = { action: 'saveSearch', title: $gettext('Save search'), value: searchQuery.value.trim() }
}

function createFolder() {
  textPrompt.value = { action: 'newFolder', title: $gettext('New folder'), value: '' }
}

function createSubfolder(entry: NavEntry) {
  openMenu.value = ''
  textPrompt.value = { action: 'newSubfolder', title: $gettext('New subfolder'), value: '', entryKey: entry.key }
}

async function confirmTextPrompt() {
  const p = textPrompt.value
  if (!p || !p.value.trim()) return
  textPromptSaving.value = true
  try {
    if (p.action === 'newFolder') {
      await api.addFolder(p.value.trim())
    } else if (p.action === 'newSubfolder') {
      const fo = folders.value.find((x) => `folder-${x.id}` === p.entryKey)
      await api.addFolder(p.value.trim(), fo?.id ?? null)
    } else if (p.action === 'renameFeed') {
      const f = feeds.value.find((x) => `feed-${x.id}` === p.entryKey)
      if (f) await api.renameFeed(f.id, p.value.trim())
    } else if (p.action === 'renameFolder') {
      const fo = folders.value.find((x) => `folder-${x.id}` === p.entryKey)
      if (fo) await api.renameFolder(fo.id, p.value.trim())
    } else {
      await api.addSearch(p.value.trim(), searchQuery.value.trim())
    }
    textPrompt.value = null
    await loadSidebar()
  } catch (e) {
    error.value = errText(e)
  } finally {
    textPromptSaving.value = false
  }
}

function entryFeed(entry: NavEntry): Feed | undefined {
  return feeds.value.find((f) => `feed-${f.id}` === entry.key)
}
function entryFolder(entry: NavEntry): Folder | undefined {
  return folders.value.find((f) => `folder-${f.id}` === entry.key)
}

function renameFeed(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  textPrompt.value = { action: 'renameFeed', title: $gettext('Rename'), value: f.title, entryKey: entry.key }
}

// Mover a carpeta con selector (antes: window.prompt con el nombre a mano).
const moveOpen = ref(false)
const moveFeedId = ref(0)
const moveFolderSel = ref(0)
const moveSaving = ref(false)

function moveFeed(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  moveFeedId.value = f.id
  moveFolderSel.value = f.folderId ?? 0
  moveOpen.value = true
}

async function confirmMove() {
  moveSaving.value = true
  try {
    await api.moveFeed(moveFeedId.value, moveFolderSel.value || null)
    moveOpen.value = false
    await loadSidebar()
  } catch (e) {
    error.value = errText(e)
  } finally {
    moveSaving.value = false
  }
}

function renameFolder(entry: NavEntry) {
  openMenu.value = ''
  const fo = entryFolder(entry)
  if (!fo) return
  textPrompt.value = { action: 'renameFolder', title: $gettext('Rename'), value: fo.name, entryKey: entry.key }
}

// Credenciales Basic del feed (#27).
const credOpen = ref(false)
const credFeedId = ref(0)
const credUser = ref('')
const credPass = ref('')
const credHasAuth = ref(false)
const credSaving = ref(false)

function openCredentials(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  credFeedId.value = f.id
  credUser.value = f.authUser || ''
  credPass.value = ''
  credHasAuth.value = !!f.authUser
  credOpen.value = true
}

async function saveCredentials() {
  if (!credUser.value.trim()) {
    error.value = $gettext('Username is required to set credentials')
    return
  }
  credSaving.value = true
  try {
    await api.setFeedCredentials(credFeedId.value, credUser.value.trim(), credPass.value)
    credOpen.value = false
    await loadSidebar()
  } catch (e) {
    error.value = extractErrorMessage(e, $gettext('Could not save credentials'))
  } finally {
    credSaving.value = false
  }
}

async function removeCredentials() {
  credSaving.value = true
  try {
    await api.setFeedCredentials(credFeedId.value, '', '')
    credOpen.value = false
    await loadSidebar()
  } catch (e) {
    error.value = extractErrorMessage(e, $gettext('Could not save credentials'))
  } finally {
    credSaving.value = false
  }
}

function openOriginal() {
  if (detail.value?.url) window.open(detail.value.url, '_blank', 'noopener,noreferrer')
}

// Compartir artículo con URL pública (#43): crea el token, copia el enlace y
// permite revocarlo con un segundo clic.
async function toggleShare() {
  if (!detail.value) return
  if (sharedUrl.value) {
    try {
      await api.unshareItem(detail.value.id)
      sharedUrl.value = ''
    } catch (e) {
      error.value = $gettext('Could not stop sharing: ') + errText(e)
    }
    return
  }
  try {
    const res = await api.shareItem(detail.value.id)
    const url = window.location.origin + res.share.url
    sharedUrl.value = url
    try {
      await navigator.clipboard.writeText(url)
    } catch {
      /* sin permiso de portapapeles: se muestra el enlace como aviso */
    }
    error.value = $gettext('Share link copied')
  } catch (e) {
    error.value = $gettext('Could not share article: ') + errText(e)
  }
}

async function markEntryRead(entry: NavEntry) {
  openMenu.value = ''
  const newest = await currentNewest()
  if (entry.kind === 'feed') await api.markFeedRead((entry.sel as { id: number }).id, newest)
  if (entry.kind === 'folder') await api.markFolderRead((entry.sel as { id: number }).id, newest)
  await loadSidebar()
  await loadItems()
}

async function currentNewest(): Promise<number> {
  const f = await api.feeds()
  return f.newestItemId ?? 0
}

async function deleteEntry(entry: NavEntry) {
  openMenu.value = ''
  let label = entry.label
  if (entry.kind === 'feed') label = entryFeed(entry)?.title ?? entry.label
  else if (entry.kind === 'folder') label = entryFolder(entry)?.name ?? entry.label
  const extra = entry.kind === 'folder' ? ' ' + $gettext('(feeds inside will be deleted)') : ''
  if (!window.confirm($gettext('Delete') + ` "${label}"?` + extra)) return
  try {
    if (entry.kind === 'feed') await api.deleteFeed((entry.sel as { id: number }).id)
    else if (entry.kind === 'folder') await api.deleteFolder((entry.sel as { id: number }).id)
    else if (entry.kind === 'search' && entry.searchId != null) {
      await api.deleteSearch(entry.searchId)
      if (searchActive.value) {
        searchQuery.value = ''
        detail.value = null
      }
    }
    if (isActive(entry)) router.push('/news/')
    await loadSidebar()
    await loadItems()
  } catch (e) {
    error.value = $gettext('Delete failed: ') + errText(e)
  }
}

async function openFilter(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  filterFeedId.value = f.id
  filterForm.value = { titleKeywords: '', bodyKeywords: '', urlKeywords: '' }
  try {
    const res = await api.getFilter(f.id)
    filterForm.value = {
      titleKeywords: res.filter.titleKeywords ?? '',
      bodyKeywords: res.filter.bodyKeywords ?? '',
      urlKeywords: res.filter.urlKeywords ?? ''
    }
  } catch {
    /* sin filtro previo: campos vacíos */
  }
  filterOpen.value = true
}

function closeFilter() {
  filterOpen.value = false
  filterFeedId.value = null
}

async function saveFilter() {
  if (filterFeedId.value == null) return
  filterSaving.value = true
  try {
    await api.setFilter(filterFeedId.value, {
      titleKeywords: filterForm.value.titleKeywords.trim(),
      bodyKeywords: filterForm.value.bodyKeywords.trim(),
      urlKeywords: filterForm.value.urlKeywords.trim()
    })
    closeFilter()
  } catch (e) {
    error.value = $gettext('Could not save filter: ') + errText(e)
  } finally {
    filterSaving.value = false
  }
}

async function clearFilter() {
  if (filterFeedId.value == null) return
  filterSaving.value = true
  try {
    await api.deleteFilter(filterFeedId.value)
    closeFilter()
  } catch (e) {
    error.value = $gettext('Could not clear filter: ') + errText(e)
  } finally {
    filterSaving.value = false
  }
}

// Reglas block/keep por feed o globales (#35).
const rulesOpen = ref(false)
const rulesSaving = ref(false)
const rulesFeedId = ref<number | null>(null)
const rulesTitle = ref('')
const rulesForm = ref<Rules>({ block: '', keep: '' })

const rulesHasRules = computed(() => rulesForm.value.block.trim() !== '' || rulesForm.value.keep.trim() !== '')

async function openRules(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  rulesFeedId.value = f.id
  rulesTitle.value = $gettext('Rules for this feed')
  rulesForm.value = { block: '', keep: '' }
  try {
    const res = await api.getFeedRules(f.id)
    rulesForm.value = { block: res.rules.block ?? '', keep: res.rules.keep ?? '' }
  } catch {
    /* sin reglas previas */
  }
  rulesOpen.value = true
}

async function openGlobalRules() {
  rulesFeedId.value = null
  rulesTitle.value = $gettext('Global rules')
  rulesForm.value = { block: '', keep: '' }
  try {
    const r = await api.myRules()
    rulesForm.value = { block: r.block ?? '', keep: r.keep ?? '' }
  } catch {
    /* sin reglas previas */
  }
  rulesOpen.value = true
}

async function saveRules() {
  rulesSaving.value = true
  try {
    const payload = { block: rulesForm.value.block.trim(), keep: rulesForm.value.keep.trim() }
    if (rulesFeedId.value != null) await api.setFeedRules(rulesFeedId.value, payload)
    else await api.updateMyRules(payload)
    rulesOpen.value = false
  } catch (e) {
    error.value = $gettext('Could not save rules: ') + errText(e)
  } finally {
    rulesSaving.value = false
  }
}

async function clearRules() {
  rulesSaving.value = true
  try {
    if (rulesFeedId.value != null) await api.deleteFeedRules(rulesFeedId.value)
    else await api.updateMyRules({ block: '', keep: '' })
    rulesOpen.value = false
  } catch (e) {
    error.value = $gettext('Could not clear rules: ') + errText(e)
  } finally {
    rulesSaving.value = false
  }
}

// Reglas de auto-marcado como leído (#40).
const autoReadOpen = ref(false)
const autoReadRules = ref<AutoReadRule[]>([])
const autoReadPattern = ref('')
const autoReadFeedSel = ref(0)
const autoReadSaving = ref(false)

async function openAutoRead() {
  autoReadPattern.value = ''
  autoReadFeedSel.value = 0
  try {
    const res = await api.autoRead()
    autoReadRules.value = res.rules
  } catch {
    autoReadRules.value = []
  }
  autoReadOpen.value = true
}

function autoReadFeedLabel(feedId: number): string {
  if (feedId === 0) return $gettext('All feeds') + ' · '
  const f = feeds.value.find((x) => x.id === feedId)
  return (f?.title ?? `#${feedId}`) + ' · '
}

async function addAutoReadRule() {
  const pat = autoReadPattern.value.trim()
  if (!pat) return
  autoReadSaving.value = true
  try {
    await api.addAutoRead(autoReadFeedSel.value, pat)
    autoReadPattern.value = ''
    const res = await api.autoRead()
    autoReadRules.value = res.rules
  } catch (e) {
    error.value = $gettext('Could not add auto-read rule: ') + errText(e)
  } finally {
    autoReadSaving.value = false
  }
}

async function removeAutoRead(rule: AutoReadRule) {
  try {
    await api.deleteAutoRead(rule.id)
    autoReadRules.value = autoReadRules.value.filter((x) => x.id !== rule.id)
  } catch (e) {
    error.value = $gettext('Could not delete rule: ') + errText(e)
  }
}

// Selector CSS de extracción por feed (#39).
const scraperOpen = ref(false)
const scraperSaving = ref(false)
const scraperFeedId = ref<number | null>(null)
const scraperForm = ref('')

async function openScraper(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  scraperFeedId.value = f.id
  scraperForm.value = ''
  try {
    const res = await api.getFeedScraper(f.id)
    scraperForm.value = res.scraperSelector ?? ''
  } catch {
    /* sin selector previo */
  }
  scraperOpen.value = true
}

async function saveScraper() {
  if (scraperFeedId.value == null) return
  scraperSaving.value = true
  try {
    await api.setFeedScraper(scraperFeedId.value, scraperForm.value.trim())
    scraperOpen.value = false
  } catch (e) {
    error.value = $gettext('Could not save scraper: ') + errText(e)
  } finally {
    scraperSaving.value = false
  }
}

async function openRetention(entry: NavEntry) {  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  retentionFeedId.value = f.id
  retentionForm.value = 0
  try {
    const res = await api.getRetention(f.id)
    retentionForm.value = res.retentionDays ?? 0
  } catch {
    /* usar 0 */
  }
  retentionOpen.value = true
}

async function saveRetention() {
  if (retentionFeedId.value == null) return
  retentionSaving.value = true
  try {
    const days = Math.max(0, Math.min(3650, retentionForm.value || 0))
    await api.setRetention(retentionFeedId.value, days)
    retentionOpen.value = false
  } catch (e) {
    error.value = $gettext('Could not save retention: ') + errText(e)
  } finally {
    retentionSaving.value = false
  }
}

async function openSettings() {
  settingsOpen.value = true
  try {
    const s = await api.mySettings()
    settingsForm.value = {
      theme: s.theme || 'system',
      readerMaxWidth: s.readerMaxWidth || 'wide',
      feedIntervalMin: s.feedIntervalMin ?? '',
      readerFont: s.readerFont || 'default',
      readerFontSize: s.readerFontSize || '15',
      ntfyTopic: s.ntfyTopic ?? ''
    }
  } catch {
    /* usar defaults */
  }
}

async function saveSettings() {
  settingsSaving.value = true
  try {
    const updated = await api.updateSettings({
      theme: settingsForm.value.theme,
      readerMaxWidth: settingsForm.value.readerMaxWidth,
      feedIntervalMin: settingsForm.value.feedIntervalMin.trim(),
      readerFont: settingsForm.value.readerFont,
      readerFontSize: settingsForm.value.readerFontSize,
      ntfyTopic: settingsForm.value.ntfyTopic?.trim() ?? ''
    })
    userTheme.value = updated.theme || 'system'
    userWidth.value = updated.readerMaxWidth || 'wide'
    userFont.value = updated.readerFont || 'default'
    userFontSize.value = updated.readerFontSize || '15'
    settingsOpen.value = false
  } catch (e) {
    error.value = $gettext('Could not save settings: ') + errText(e)
  } finally {
    settingsSaving.value = false
  }
}

async function exportOpml() {
  try {
    const res = await api.exportOpml()
    const blob = new Blob([res.data as string], { type: 'text/x-opml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'subscriptions.opml'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    error.value = $gettext('Export failed: ') + errText(e)
  }
}

async function importOpml(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const res: { data: { imported: number; skipped: number } } = await api.importOpml(file)
    error.value = ''
    await loadSidebar()
    await refreshNow()
    window.alert($gettext('Imported') + `: ${res.data.imported}, ` + $gettext('skipped') + `: ${res.data.skipped}`)
  } catch (e) {
    error.value = $gettext('Import failed: ') + errText(e)
  } finally {
    input.value = ''
  }
}

function errText(e: unknown): string {
  const anyE = e as { response?: { status?: number; data?: unknown }; message?: string }
  if (anyE?.response?.status) return `HTTP ${anyE.response.status}`
  return anyE?.message ?? String(e)
}

function extractErrorMessage(e: unknown, fallback: string): string {
  const anyE = e as { response?: { status?: number } }
  if (anyE?.response?.status === 409) return $gettext('Already subscribed')
  if (anyE?.response?.status === 422) return $gettext('Feed not readable')
  return fallback
}

watch(selection, () => loadItems())
watch(showAll, () => loadItems())
watch(oldestFirst, () => loadItems())
watch(() => route.value.fullPath, () => (detail.value = null))

// Favicon de la app: al abrir News ponemos el icono de la extensión en la
// pestaña; al salir restauramos el del host. El host aplica el favicon por
// tema (link[rel~='icon']), así que guardamos el original al montar.
let savedFavicon = ''

// newsFaviconURL deriva la URL del asset desde la del bundle JS
// (/assets/apps/news/js/*.mjs -> /assets/apps/news/news-favicon.svg).
function newsFaviconURL(): string {
  try {
    const u = new URL(import.meta.url)
    const parts = u.pathname.split('/')
    const assetsIdx = parts.lastIndexOf('assets')
    if (assetsIdx >= 0) {
      const base = parts.slice(0, assetsIdx + 2).join('/') // .../assets/<appId>
      return `${u.origin}${base}/news-favicon.svg`
    }
  } catch {
    /* fallthrough */
  }
  return '/assets/apps/news/news-favicon.svg'
}

function applyNewsFavicon() {
  const link: HTMLLinkElement | null = document.querySelector("link[rel~='icon']")
  if (link) {
    savedFavicon = link.href
    link.href = newsFaviconURL()
  } else {
    const l = document.createElement('link')
    l.rel = 'icon'
    l.href = newsFaviconURL()
    document.head.appendChild(l)
  }
}

function restoreHostFavicon() {
  if (!savedFavicon) return
  const link: HTMLLinkElement | null = document.querySelector("link[rel~='icon']")
  if (link) link.href = savedFavicon
}

onMounted(async () => {
  applyNewsFavicon()
  mediaDark.addEventListener('change', onMediaChange)
  window.addEventListener('keydown', onKeydown)
  try {
    await loadSettings()
    await loadSidebar()
  } catch (e) {
    error.value = $gettext('Could not load folders/feeds: ') + errText(e)
  }
  await loadItems()
})

onUnmounted(() => {
  restoreHostFavicon()
  mediaDark.removeEventListener('change', onMediaChange)
  window.removeEventListener('keydown', onKeydown)
  resizeEnd()
})
</script>

<style>
main {
  /* Variables de tema; se sobreescriben en .news-theme-dark */
  --news-bg: #ffffff;
  --news-fg: #1a1a1a;
  --news-border: rgba(125, 125, 125, 0.25);
  --news-border-light: rgba(125, 125, 125, 0.2);
  --news-border-lighter: rgba(125, 125, 125, 0.15);
  --news-border-medium: rgba(125, 125, 125, 0.3);
  --news-border-soft: rgba(125, 125, 125, 0.35);
  --news-input-border: #94a3b8;
  --news-primary: #2563eb;
  --news-warning: #d97706;
  --news-error: #b91c1c;
  --news-muted: rgba(0, 0, 0, 0.35);
  --news-active-bg: rgba(0, 100, 200, 0.12);
  --news-hover-bg: rgba(0, 100, 200, 0.08);
  --news-shadow: rgba(0, 0, 0, 0.15);
}
/* The host sets global element rules (body, h1-h6, p, dialog, a, a:hover)
   and oc-button colors from its --oc-role-* tokens, and a direct rule beats
   inheritance. Override those tokens in both themes so the app palette does
   not depend on the host theme (issue #23). */
main.news-theme-light {
  --oc-role-surface: var(--news-bg);
  --oc-role-on-surface: var(--news-fg);
  --oc-role-secondary: #4b5563;
  --oc-role-on-secondary: #ffffff;
}
main.news-theme-dark {
  --news-bg: #161616;
  --news-fg: #e5e5e5;
  --news-border: rgba(255, 255, 255, 0.18);
  --news-border-light: rgba(255, 255, 255, 0.12);
  --news-border-lighter: rgba(255, 255, 255, 0.08);
  --news-border-medium: rgba(255, 255, 255, 0.22);
  --news-border-soft: rgba(255, 255, 255, 0.3);
  --news-input-border: #64748b;
  --news-primary: #60a5fa;
  --news-warning: #f59e0b;
  --news-error: #f87171;
  --news-muted: rgba(255, 255, 255, 0.4);
  --news-active-bg: rgba(96, 165, 250, 0.18);
  --news-hover-bg: rgba(96, 165, 250, 0.12);
  --news-shadow: rgba(0, 0, 0, 0.45);
  --oc-role-surface: var(--news-bg);
  --oc-role-on-surface: var(--news-fg);
  --oc-role-secondary: #cbd5e1;
  --oc-role-on-secondary: #1a1a1a;
}
@keyframes news-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
/* Menú lateral: mismos valores que el sidebar de ocnotes y que el del host
   (230px, fondo surface-container #f6f8fa, filas de 14px, radio 8px, iconos
   16px, hover surface-container-highest, activo secondary-container). */
.news-sidebar {
  display: flex;
  flex-direction: column;
  width: 230px;
  flex-shrink: 0;
  background: var(--oc-role-surface-container);
  border-right: 1px solid var(--oc-role-outline-variant);
  overflow: hidden;
}
.news-sidebar-top {
  padding: 8px;
}
.news-sidebar-nav {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
.news-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 8px 4px;
}
.news-section-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--oc-role-on-surface-variant);
}
.news-nav-entry {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 8px;
  border-radius: 10.5px;
  cursor: pointer;
  font-size: 14px;
  color: var(--oc-role-on-surface);
}
.news-sidebar-bottom {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px;
  border-top: 1px solid var(--oc-role-outline-variant);
}
.news-quiet-action {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 8px;
  border: 0;
  border-radius: 8px;
  background: none;
  color: var(--oc-role-on-surface);
  font-family: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}
.news-quiet-action:hover {
  background: var(--oc-role-surface-container-highest);
}
/* hover de navegación (carpetas/suscripciones) y de opciones del menú (#48):
   mismo token que el menú lateral nativo del host (SidebarNavItem.vue) */
.news-nav-entry:hover {
  background: var(--oc-role-surface-container-highest);
}
.news-menu-btn:hover {
  background: var(--oc-role-surface-container-highest);
}
/* Botones CTA (ajustes + añadir feed): mismo aspecto que el botón flotante
   del host en OpenCloud 9 (web-runtime SidebarNav.vue: appearance="filled"
   color-role="primary"), sin gradiente propio. El color sale del token del
   tema, así que sigue al tema claro/oscuro sin hex fijos. (#49) */
.news-cta {
  appearance: none;
  border: 0;
  margin: 0;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  border-radius: 4px;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  background: var(--oc-role-primary);
  color: var(--oc-role-on-primary);
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1);
  transition: filter 150ms ease-out, box-shadow 150ms ease-out;
}
.news-cta:hover {
  filter: brightness(1.1);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1);
}
.news-cta:disabled {
  opacity: 0.5;
  cursor: default;
  filter: none;
}
main.news-theme-dark .news-cta {
  background: var(--oc-role-primary);
  color: var(--oc-role-on-primary);
}
.news-body p { margin: 0 0 0.9em; line-height: 1.65; }
.news-body h1, .news-body h2, .news-body h3, .news-body h4 {
  margin: 1.2em 0 0.5em;
  line-height: 1.3;
  font-weight: 600;
}
.news-body h2 { font-size: 1.3em; }
.news-body h3 { font-size: 1.15em; }
.news-body ul, .news-body ol { margin: 0 0 1em; padding-left: 1.4em; line-height: 1.6; }
.news-body li { margin-bottom: 0.3em; }
.news-body img {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
  margin: 0.4em 0 1em;
}
.news-body a { color: var(--news-primary); text-decoration: underline; }
.news-body a:hover { color: var(--news-primary); }
.news-body blockquote {
  margin: 0 0 1em;
  padding: 0.4em 1em;
  border-left: 3px solid var(--news-border-soft);
  opacity: 0.9;
}
.news-body pre {
  background: var(--news-border-lighter);
  border-radius: 8px;
  padding: 0.8em 1em;
  overflow-x: auto;
  font-size: 0.9em;
}
.news-body code { background: var(--news-border-lighter); border-radius: 4px; padding: 0.1em 0.35em; }
.news-body pre code { background: none; padding: 0; }
.news-body figure { margin: 0 0 1em; }
.news-body figcaption { font-size: 0.85em; opacity: 0.65; margin-top: 0.3em; }
.news-body table { border-collapse: collapse; margin-bottom: 1em; }
.news-body th, .news-body td { border: 1px solid var(--news-border-soft); padding: 0.4em 0.7em; }
</style>
