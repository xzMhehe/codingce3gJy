<template>
  <div class="farm-admin">
    <el-tabs v-model="tab" type="card" @tab-click="onTab">
      <!-- ============ 帮派管理 ============ -->
      <el-tab-pane label="帮派管理" name="gangs">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-input v-model="gangWord" placeholder="帮派名 / 帮主搜索" clearable style="width:220px"
                      @keyup.enter.native="gangPage = 1; loadGangs()" />
            <el-button type="primary" icon="el-icon-search" @click="gangPage = 1; loadGangs()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="gangs" v-loading="gangLoading" stripe border>
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column label="帮派名" min-width="130" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="70" align="center" />
            <el-table-column prop="exp" label="经验" width="90" align="center" />
            <el-table-column label="帮主" min-width="110" show-overflow-tooltip>
              <template slot-scope="{row}">{{ row.master || '—' }}</template>
            </el-table-column>
            <el-table-column prop="members" label="成员" width="70" align="center" />
            <el-table-column label="公告" min-width="160" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-sub">{{ row.notice || '—' }}</span></template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="160" />
            <el-table-column label="操作" width="90" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="dissolve(row)">解散</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination background layout="total, sizes, prev, pager, next" :total="gangTotal"
                         :page-size="gangSize" :current-page="gangPage" :page-sizes="[10, 20, 50]"
                         @current-change="p => { gangPage = p; loadGangs() }"
                         @size-change="s => { gangSize = s; gangPage = 1; loadGangs() }" />
        </el-card>
      </el-tab-pane>

      <!-- ============ 聊天记录 ============ -->
      <el-tab-pane label="聊天记录" name="chats">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-input v-model="chatWord" placeholder="家园号 / 昵称 / 内容搜索" clearable style="width:240px"
                      @keyup.enter.native="chatPage = 1; loadChats()" />
            <el-button type="primary" icon="el-icon-search" @click="chatPage = 1; loadChats()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="chats" v-loading="chatLoading" stripe border>
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="user_id" label="家园号" width="85" align="center" />
            <el-table-column label="昵称" width="110" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.nick || '—' }}</span></template>
            </el-table-column>
            <el-table-column label="内容" min-width="240" show-overflow-tooltip>
              <template slot-scope="{row}">{{ row.content }}</template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" width="150" />
            <el-table-column label="操作" width="90" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delChat(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination background layout="total, sizes, prev, pager, next" :total="chatTotal"
                         :page-size="chatSize" :current-page="chatPage" :page-sizes="[20, 50, 100]"
                         @current-change="p => { chatPage = p; loadChats() }"
                         @size-change="s => { chatSize = s; chatPage = 1; loadChats() }" />
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminJwtData',
  data () {
    return {
      tab: 'gangs',
      // 帮派
      gangs: [], gangTotal: 0, gangPage: 1, gangSize: 10, gangLoading: false, gangWord: '',
      // 聊天
      chats: [], chatTotal: 0, chatPage: 1, chatSize: 20, chatLoading: false, chatWord: ''
    }
  },
  mounted () { this.loadGangs() },
  methods: {
    onTab () {
      if (this.tab === 'gangs' && this.gangs.length === 0) this.loadGangs()
      if (this.tab === 'chats' && this.chats.length === 0) this.loadChats()
    },
    loadGangs () {
      this.gangLoading = true
      api.get('/admin/jwt-gangs', { params: { page: this.gangPage, size: this.gangSize, word: this.gangWord } }).then(r => {
        this.gangLoading = false
        if (r.code === 0) {
          this.gangs = r.data.list
          this.gangTotal = r.data.total
          this.gangPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    dissolve (row) {
      this.$confirm('解散帮派将清除所有成员与申请记录，确认解散「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/jwt-gangs/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已解散'); this.loadGangs() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadChats () {
      this.chatLoading = true
      api.get('/admin/jwt-chats', { params: { page: this.chatPage, size: this.chatSize, word: this.chatWord } }).then(r => {
        this.chatLoading = false
        if (r.code === 0) {
          this.chats = r.data.list
          this.chatTotal = r.data.total
          this.chatPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    delChat (row) {
      this.$confirm('确认删除该条聊天记录？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/jwt-chats/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadChats() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
