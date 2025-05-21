# Nakamoto Bonus Demo

The `NakamotoBonus.ipynb` demo showcases the effect of assigning the block reward in a proportional and in a uniform way. The user is able to change using a widget the state of the delegations on the network and amount of the reward that is assigned proportionally vs uniformly and see how this affects the way reward is split across validators. 
It can be seen that, increasing the amount of reward that is assigned uniformly - i.e. the Nakamoto Bonus - increases the incentive of delegators to delegate on validators having less voting power.

## Requirements

The file `NakamotoBonus.ipynb` is a Jupyter notebook file and it requires python and jupyterlab to be installed in the system. There are several ways to install Jupyter lab, for more information refer to this [link](https://jupyterlab.readthedocs.io/en/stable/getting_started/installation.html).

## Usage

Jupyter should be launched using the terminal from the folder containing the file `NakamotoBonus.ipynb`.

```bash
jupyter lab
```

This will launch the jupyter webui. The user can then select the `NakamotoBonus.ipynb` using the file browser on the left side of the UI, and execute the code in the cells to generate the charts.
