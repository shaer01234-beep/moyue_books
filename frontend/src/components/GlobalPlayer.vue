<template>
  <div v-if="playerState.task" class="global-player">
    <!-- 底部迷你播放条（音乐播放器模式） -->
    <div class="gp-bar" :class="{ open: playerState.panelVisible }" @click="openPanel">
      <div class="gp-cover" :style="{ background: coverGradient }">{{ coverText }}</div>
      <div class="gp-info">
        <div class="gp-title">
          {{ playerState.task.book_title }}
          <span v-if="playerState.waitingNext" class="gp-waiting">续播中…</span>
        </div>
        <div class="gp-sub">{{ currentSegTitle }}</div>
      </div>
      <div class="gp-progress" @click.stop="onSeekClick">
        <div class="gp-pbar"><div class="gp-fill" :style="{ width: progressWidth + '%' }"></div></div>
      </div>
      <div class="gp-time">{{ currentTimeStr }} / {{ durationStr }}</div>
      <div class="gp-actions" @click.stop>
        <button class="gp-ctrl" title="播放/暂停" @click="togglePlay">
          <el-icon :size="20"><component :is="playerState.playing ? 'VideoPause' : 'VideoPlay'" /></el-icon>
        </button>
        <button class="gp-ctrl" title="下一段" @click="playNext">
          <el-icon :size="18"><Right /></el-icon>
        </button>
        <button class="gp-expand" @click="openPanel">
          <el-icon :size="16"><ArrowUp /></el-icon>
          <span>播放列表</span>
        </button>
        <button class="gp-close" title="关闭播放器" @click="stopPlayer">
          <el-icon :size="16"><Close /></el-icon>
        </button>
      </div>
    </div>

    <!-- 播放列表弹层（独立可复用组件） -->
    <transition name="gp-mask">
      <AudioPlaylist
        v-if="playerState.panelVisible"
        :segments="playerState.segments"
        :current-index="playerState.index"
        :book-title="playerState.task ? playerState.task.book_title : ''"
        :cover-text="coverText"
        :cover-gradient="coverGradient"
        :is-generating="isGen"
        :waiting-next="playerState.waitingNext"
        @play-segment="playSegment"
        @close="closePanel"
      />
    </transition>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  playerState,
  currentSegTitle,
  progressWidth,
  currentTimeStr,
  durationStr,
  togglePlay,
  playNext,
  playSegment,
  seekByClick,
  closePanel,
  stopPlayer,
} from '@/player'
import AudioPlaylist from '@/components/AudioPlaylist.vue'

// 当前任务是否处于生成中（用于播放列表状态展示）
const isGen = computed(() => playerState.task &&
  (playerState.task.status === 'processing' || playerState.task.status === 'pending'))

const coverText = computed(() => (playerState.task?.book_title || '声').trim().charAt(0).toUpperCase())

const coverGradient = computed(() => {
  const palettes = [
    'linear-gradient(135deg,#6366f1,#8b5cf6)',
    'linear-gradient(135deg,#06b6d4,#22d3ee)',
    'linear-gradient(135deg,#10b981,#34d399)',
    'linear-gradient(135deg,#f59e0b,#fbbf24)',
  ]
  const id = playerState.task?.id || 0
  return palettes[id % palettes.length]
})

function openPanel() {
  playerState.panelVisible = true
}

function onSeekClick(e) {
  seekByClick(e, e.currentTarget.querySelector('.gp-pbar'))
}
</script>

<style scoped>
.global-player {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 3000;
  pointer-events: none;
}
.global-player > * {
  pointer-events: auto;
}

/* ===== 迷你播放条 ===== */
.gp-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 58px;
  margin: 0 auto 14px;
  width: min(760px, calc(100% - 24px));
  padding: 0 14px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 1px solid rgba(99, 102, 241, 0.16);
  box-shadow: 0 10px 32px rgba(30, 41, 59, 0.16);
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}
.gp-bar:hover {
  box-shadow: 0 12px 36px rgba(99, 102, 241, 0.22);
}
.gp-bar.open {
  opacity: 0;
  transform: translateY(8px);
  pointer-events: none;
}

.gp-cover {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: 700;
}

.gp-info {
  flex-shrink: 0;
  min-width: 0;
  max-width: 220px;
}
.gp-title {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  gap: 6px;
}
.gp-sub {
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 1px;
}
.gp-waiting {
  flex-shrink: 0;
  font-size: 11px;
  color: #6366f1;
  background: #eef2ff;
  border-radius: 6px;
  padding: 1px 6px;
}

.gp-progress {
  flex: 1;
  min-width: 60px;
  cursor: pointer;
}
.gp-pbar {
  height: 5px;
  border-radius: 3px;
  background: #e2e8f0;
  overflow: hidden;
}
.gp-pbar.big {
  height: 6px;
}
.gp-fill {
  height: 100%;
  border-radius: 3px;
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  transition: width 0.2s linear;
}

.gp-time {
  flex-shrink: 0;
  font-size: 12px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
}

.gp-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}
.gp-ctrl,
.gp-expand,
.gp-close {
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  cursor: pointer;
  color: #475569;
  border-radius: 10px;
  transition: background 0.15s, color 0.15s, transform 0.15s;
  height: 32px;
}
.gp-ctrl {
  width: 32px;
}
.gp-ctrl:hover {
  color: var(--app-primary, #6366f1);
  background: #eef2ff;
}
.gp-ctrl:active {
  transform: scale(0.9);
}
.gp-expand {
  gap: 4px;
  padding: 0 10px;
  font-size: 12px;
  color: #6366f1;
  background: #eef2ff;
}
.gp-expand:hover {
  background: #e0e7ff;
}
.gp-close {
  width: 30px;
  color: #94a3b8;
}
.gp-close:hover {
  color: #ef4444;
  background: #fef2f2;
}

/* ===== 移动端适配 ===== */
@media (max-width: 640px) {
  .global-player {
    bottom: 62px; /* 底部导航上方 */
  }
  /* 迷你条：还原完整功能（封面/标题/播放/下一段/播放列表/关闭），居中收窄更协调 */
  .gp-bar {
    width: calc(100% - 24px);
    max-width: 420px;
    margin: 0 auto 8px;
    gap: 8px;
    padding: 0 10px;
    height: 54px;
    border-radius: 14px;
    box-shadow: 0 8px 28px rgba(15, 23, 42, 0.16);
  }
  .gp-cover {
    width: 36px;
    height: 36px;
    font-size: 16px;
    border-radius: 9px;
  }
  .gp-info {
    flex: 1;
    min-width: 0;
    max-width: none;
  }
  .gp-title {
    font-size: 13px;
  }
  .gp-sub {
    font-size: 11px;
  }
  /* 迷你条上的进度条在手机上精度不足，隐藏；精确 seek 走详情面板 */
  .gp-progress,
  .gp-time {
    display: none;
  }
  .gp-actions {
    gap: 2px;
  }
  .gp-ctrl {
    width: 30px;
    height: 30px;
  }
  .gp-expand span {
    display: none;
  }
  .gp-expand {
    width: 30px;
    padding: 0;
  }
  .gp-close {
    width: 28px;
    height: 30px;
  }
}
</style>
