import { VuexModule, Module, Action, getModule } from "vuex-module-decorators";
import store from "@/store";
import Vue from "vue";
import { API_ENDPOINT } from "../../constants";
import { GatewayError } from "../../types";

export interface Group {
  name: string;
}

export interface GroupList {
  groups: Group[];
}

@Module({ dynamic: true, store, name: "groups" })
class GroupMod extends VuexModule {
  @Action({ rawError: true })
  public async newGroup(group: Group): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      Vue.axios.put(API_ENDPOINT + "/group", group).then(
        () => {
          resolve();
        },
        err => {
          reject(err.response?.data as GatewayError);
        }
      );
    });
  }

  @Action({ rawError: true })
  public async deleteGroup(name: string): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      Vue.axios.delete(API_ENDPOINT + "/group/" + name, {}).then(
        () => {
          resolve();
        },
        err => {
          reject(err.response?.data as GatewayError);
        }
      );
    });
  }

  @Action({ rawError: true })
  public async renameGroup(name: string): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      Vue.axios.post(API_ENDPOINT + "/group/rename/" + name, {}).then(
        () => {
          resolve();
        },
        err => {
          reject(err.response?.data as GatewayError);
        }
      );
    });
  }

  @Action({ rawError: true })
  public async getGroups(req: {
    page: number;
    perPage: number;
    search: string;
  }): Promise<GroupList> {
    // we will not configure sort_key or sort_order
    const request: { start?: number; end?: number; filters?: string } = {
      start: (req.page - 1) * req.perPage,
      end: req.page * req.perPage
    };
    if (req.search.length > 0) {
      request.filters = `(cn=*${req.search}*)`;
    }
    return new Promise<GroupList>((resolve, reject) => {
      Vue.axios.get(API_ENDPOINT + "/groups", { params: request }).then(
        response => {
          resolve(response.data);
        },
        err => {
          reject(err.response?.data as GatewayError);
        }
      );
    });
  }
}

export const GroupModule = getModule(GroupMod);
