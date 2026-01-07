<template>
    <v-container fluid>
        <v-row>
            <v-col cols="12">
                <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog">新增</v-btn>
            </v-col>
        </v-row>

        <v-row>
            <v-col cols="12">
               <v-data-table :headers="headers" :items="accountMaps" :loading="loading" class="elevation-1"
                    :items-per-page="15">
                    <template #item.index="{ index }">
                        {{ index + 1 }}
                    </template>

                    <template #item.actions="{ item }">
                        <v-btn icon="mdi-pencil" size="small" variant="text" color="primary"
                            @click="openEditDialog(item)"></v-btn>
                        <v-btn icon="mdi-delete" size="small" color="error" variant="text"
                            @click="deleteItem(item)"></v-btn>
                    </template>
                </v-data-table>
            </v-col>
        </v-row>

        <v-dialog v-model="dialog" max-width="500px" persistent>
            <v-card>
                <v-card-title class="bg-primary text-white px-4 py-3">
                    <span class="text-h6">{{ isEditing ? '编辑' : '新增' }} Account Map</span>
                </v-card-title>

                <v-card-text class="mt-4">
                    <v-container>
                        <v-row>
                            <v-col cols="12">
                                <v-text-field v-model="form.keyword" label="Keyword" density="comfortable"
                                    variant="outlined" required></v-text-field>
                            </v-col>
                            <v-col cols="12">
                                <v-text-field v-model="form.account" label="Account" density="comfortable"
                                    variant="outlined" required></v-text-field>
                            </v-col>
                            <v-col cols="12">
                                <v-select v-model="form.type" :items="typeOptions" label="Type" density="comfortable"
                                    variant="outlined" required></v-select>
                            </v-col>
                        </v-row>
                    </v-container>
                </v-card-text>

                <v-divider></v-divider>

                <v-card-actions class="pa-4">
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="closeDialog">取消</v-btn>
                    <v-btn color="primary" variant="elevated" @click="saveItem">
                        {{ isEditing ? '更新' : '创建' }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-container>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import type { AccountMap } from '../services/BeangoService';
import { getAllAccountMaps, createAccountMap, updateAccountMap, deleteAccountMap } from '../services/BeangoService';

const accountMaps = ref<AccountMap[]>([]);
const loading = ref(false);
const dialog = ref(false);
const isEditing = ref(false);
const editingId = ref<number | null>(null);

// 下拉框选项定义
const typeOptions = ['asset', 'income', 'expense'];

const form = ref({
    keyword: '',
    account: '',
    type: ''
});

const headers: any[] = [
    { title: '序号', key: 'index', align: 'end', sortable: false },
    { title: 'Keyword', key: 'keyword', align: 'end' },
    { title: 'Account', key: 'account', align: 'end' },
    { title: 'Type', key: 'type', align: 'end' },
    { title: 'Actions', key: 'actions', sortable: false, align: 'end' }
];

const loadData = async () => {
    loading.value = true;
    try {
        accountMaps.value = await getAllAccountMaps();
    } catch (error) {
        console.error('Failed to load account maps:', error);
    } finally {
        loading.value = false;
    }
};

const openCreateDialog = () => {
    isEditing.value = false;
    editingId.value = null;
    form.value = { keyword: '', account: '', type: 'asset' }; // 默认选一个
    dialog.value = true;
};

const openEditDialog = (item: AccountMap) => {
    isEditing.value = true;
    editingId.value = item.ID;
    form.value = { keyword: item.keyword, account: item.account, type: item.type };
    dialog.value = true;
};

const closeDialog = () => {
    dialog.value = false;
};

const saveItem = async () => {
    try {
        if (isEditing.value && editingId.value) {
            await updateAccountMap(editingId.value, form.value);
        } else {
            await createAccountMap(form.value);
        }
        closeDialog();
        await loadData();
    } catch (error) {
        console.error('Failed to save:', error);
    }
};

const deleteItem = async (item: AccountMap) => {
    if (confirm('确认删除？')) {
        try {
            await deleteAccountMap(item.ID);
            await loadData();
        } catch (error) {
            console.error('Failed to delete:', error);
        }
    }
};

onMounted(() => {
    loadData();
});
</script>